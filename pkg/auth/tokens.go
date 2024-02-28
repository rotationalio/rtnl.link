package auth

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	jwt "github.com/golang-jwt/jwt/v4"
	"github.com/oklog/ulid/v2"
	"github.com/rotationalio/rtnl.link/pkg/config"
	"google.golang.org/api/idtoken"
)

// Global variables that should really not be changed except between major versions.
// NOTE: the signing method should match the value returned by the JWKS
var (
	signingMethod = jwt.SigningMethodRS256
	nilID         = ulid.ULID{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00}
)

type TokenManager struct {
	conf         config.AuthConfig
	currentKeyID ulid.ULID
	currentKey   *rsa.PrivateKey
	keys         map[ulid.ULID]*rsa.PublicKey
	kidEntropy   io.Reader
	gValidator   *idtoken.Validator
}

// New creates a TokenManager with the specified keys which should be a mapping of ULID
// strings to paths to files that contain PEM encoded RSA private keys. This input is
// specifically designed for the config environment variable so that keys can be loaded
// from k8s or vault secrets that are mounted as files on disk.
func New(conf config.AuthConfig) (tm *TokenManager, err error) {
	tm = &TokenManager{
		conf: conf,
		keys: make(map[ulid.ULID]*rsa.PublicKey),
		kidEntropy: &ulid.LockedMonotonicReader{
			MonotonicReader: ulid.Monotonic(rand.Reader, 0),
		},
	}

	// Initialize google token validator
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	if tm.gValidator, err = idtoken.NewValidator(ctx); err != nil {
		return nil, err
	}

	// Load keys from disk if specified by the configuration
	for kid, path := range conf.Keys {
		// Parse the key id
		var keyID ulid.ULID
		if keyID, err = ulid.Parse(kid); err != nil {
			return nil, fmt.Errorf("could not parse kid %q for path %s: %s", kid, path, err)
		}

		// Load the keys from disk
		var data []byte
		if data, err = os.ReadFile(path); err != nil {
			return nil, fmt.Errorf("could not read kid %s from %s: %s", kid, path, err)
		}

		var key *rsa.PrivateKey
		if key, err = jwt.ParseRSAPrivateKeyFromPEM(data); err != nil {
			return nil, fmt.Errorf("could not parse RSA private key kid %s from %s: %s", kid, path, err)
		}

		// Add the key to the key map
		tm.keys[keyID] = &key.PublicKey

		// Set the current key if it is the latest key
		if tm.currentKey == nil || keyID.Time() > tm.currentKeyID.Time() {
			tm.currentKey = key
			tm.currentKeyID = keyID
		}
	}

	// If there are no keys, generate a key to use for authentication.
	if len(tm.keys) == 0 {
		if tm.currentKey, err = rsa.GenerateKey(rand.Reader, 4096); err != nil {
			return nil, err
		}

		if tm.currentKeyID, err = tm.genKeyID(); err != nil {
			return nil, err
		}

		tm.keys[tm.currentKeyID] = &tm.currentKey.PublicKey
	}

	return tm, nil
}

func NewWithKey(key *rsa.PrivateKey, conf config.AuthConfig) (tm *TokenManager, err error) {
	tm = &TokenManager{
		conf: conf,
		keys: make(map[ulid.ULID]*rsa.PublicKey),
		kidEntropy: &ulid.LockedMonotonicReader{
			MonotonicReader: ulid.Monotonic(rand.Reader, 0),
		},
	}

	// Initialize google token validator
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	if tm.gValidator, err = idtoken.NewValidator(ctx); err != nil {
		return nil, err
	}

	var kid ulid.ULID
	if kid, err = tm.genKeyID(); err != nil {
		return nil, err
	}

	tm.keys[kid] = &key.PublicKey
	tm.currentKey = key
	tm.currentKeyID = kid
	return tm, nil
}

// Verify an access or a refresh token after parsing and return its claims.
func (tm *TokenManager) Verify(tks string) (claims *Claims, err error) {
	var token *jwt.Token
	if token, err = jwt.ParseWithClaims(tks, &Claims{}, tm.keyFunc); err != nil {
		return nil, err
	}

	var ok bool
	if claims, ok = token.Claims.(*Claims); ok && token.Valid {
		if !claims.VerifyAudience(tm.conf.Audience, true) {
			return nil, fmt.Errorf("invalid audience %q", claims.Audience)
		}

		if !claims.VerifyIssuer(tm.conf.Issuer, true) {
			return nil, fmt.Errorf("invalid issuer %q", claims.Issuer)
		}

		return claims, nil
	}

	return nil, fmt.Errorf("could not parse or verify claims from %T", token.Claims)
}

// Parse an access or refresh token verifying its signature but without verifying its
// claims. This ensures that valid JWT tokens are still accepted but claims can be
// handled on a case-by-case basis; for example by validating an expired access token
// during reauthentication.
func (tm *TokenManager) Parse(tks string) (claims *Claims, err error) {
	parser := &jwt.Parser{SkipClaimsValidation: true}
	claims = &Claims{}
	if _, err = parser.ParseWithClaims(tks, claims, tm.keyFunc); err != nil {
		return nil, err
	}
	return claims, nil
}

// Sign an access or refresh token and return the token string.
func (tm *TokenManager) Sign(token *jwt.Token) (tks string, err error) {
	// Sanity check to prevent nil panics.
	if tm.currentKey == nil || tm.currentKeyID.Compare(nilID) == 0 {
		return "", errors.New("token manager not initialized with signing keys")
	}

	// Add the kid (key id - this is the standard 3 letter JWT name) to the header.
	token.Header["kid"] = tm.currentKeyID.String()

	// Return the signed string
	return token.SignedString(tm.currentKey)
}

// CreateTokenPair returns signed access and refresh tokens for the specified claims in
// one step (since normally users want both an access and a refresh token)!
func (tm *TokenManager) CreateTokenPair(claims *Claims) (accessToken, refreshToken string, err error) {
	var atk, rtk *jwt.Token
	if atk, err = tm.CreateAccessToken(claims); err != nil {
		return "", "", fmt.Errorf("could not create access token: %w", err)
	}

	if rtk, err = tm.CreateRefreshToken(atk); err != nil {
		return "", "", fmt.Errorf("could not create refresh token: %w", err)
	}

	if accessToken, err = tm.Sign(atk); err != nil {
		return "", "", fmt.Errorf("could not sign access token: %w", err)
	}

	if refreshToken, err = tm.Sign(rtk); err != nil {
		return "", "", fmt.Errorf("could not sign refresh token: %w", err)
	}

	return accessToken, refreshToken, nil
}

// CreateToken from the claims payload without modifying the claims unless the claims
// are missing required fields that need to be updated.
func (tm *TokenManager) CreateToken(claims *Claims) *jwt.Token {
	if len(claims.Audience) == 0 {
		claims.Audience = jwt.ClaimStrings{tm.conf.Audience}
	}

	if claims.Issuer == "" {
		claims.Issuer = tm.conf.Issuer
	}
	return jwt.NewWithClaims(signingMethod, claims)
}

// CreateAccessToken from the credential payload or from an previous token if the
// access token is being reauthorized from previous credentials. Note that the returned
// token only contains the claims and is unsigned.
func (tm *TokenManager) CreateAccessToken(claims *Claims) (_ *jwt.Token, err error) {
	// Create the claims for the access token, using access token defaults.
	now := time.Now()
	sub := claims.RegisteredClaims.Subject

	var kid ulid.ULID
	if kid, err = tm.genKeyID(); err != nil {
		return nil, err
	}

	claims.RegisteredClaims = jwt.RegisteredClaims{
		ID:        strings.ToLower(kid.String()), // ID is randomly generated and shared between access and refresh tokens.
		Subject:   sub,
		Audience:  jwt.ClaimStrings{tm.conf.Audience},
		Issuer:    tm.conf.Issuer,
		IssuedAt:  jwt.NewNumericDate(now),
		NotBefore: jwt.NewNumericDate(now),
		ExpiresAt: jwt.NewNumericDate(now.Add(tm.conf.AccessDuration)),
	}
	return tm.CreateToken(claims), nil
}

// CreateRefreshToken from the Access token claims with predefined expiration. Note that
// the returned token only contains the claims and is unsigned.
func (tm *TokenManager) CreateRefreshToken(accessToken *jwt.Token) (refreshToken *jwt.Token, err error) {
	accessClaims, ok := accessToken.Claims.(*Claims)
	if !ok {
		return nil, errors.New("could not retrieve claims from access token")
	}

	// Create claims for the refresh token from the access token defaults.
	claims := &Claims{
		RegisteredClaims: jwt.RegisteredClaims{
			ID:        accessClaims.ID, // ID is randomly generated and shared between access and refresh tokens.
			Audience:  accessClaims.Audience,
			Issuer:    accessClaims.Issuer,
			Subject:   accessClaims.Subject,
			IssuedAt:  accessClaims.IssuedAt,
			NotBefore: jwt.NewNumericDate(accessClaims.ExpiresAt.Add(tm.conf.RefreshOverlap)),
			ExpiresAt: jwt.NewNumericDate(accessClaims.IssuedAt.Add(tm.conf.RefreshDuration)),
		},
	}
	return tm.CreateToken(claims), nil
}

// CurrentKey returns the ulid of the current key being used to sign tokens.
func (tm *TokenManager) CurrentKey() ulid.ULID {
	return tm.currentKeyID
}

// keyFunc is an jwt.KeyFunc that selects the RSA public key from the list of managed
// internal keys based on the kid in the token header. If the kid does not exist an
// error is returned and the token will not be able to be verified.
func (tm *TokenManager) keyFunc(token *jwt.Token) (key interface{}, err error) {
	// Per JWT security notice: do not forget to validate alg is expected
	if _, ok := token.Method.(*jwt.SigningMethodRSA); !ok {
		return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
	}

	// Fetch the kid from the header
	kid, ok := token.Header["kid"]
	if !ok {
		return nil, errors.New("token does not have kid in header")
	}

	// Parse the kid
	var keyID ulid.ULID
	if keyID, err = ulid.Parse(kid.(string)); err != nil {
		return nil, fmt.Errorf("could not parse kid: %s", err)
	}

	// Fetch the key from the list of managed keys
	if key, ok = tm.keys[keyID]; !ok {
		return nil, errors.New("unknown signing key")
	}
	return key, nil
}

func (tm *TokenManager) genKeyID() (uid ulid.ULID, err error) {
	ms := ulid.Timestamp(time.Now())
	if uid, err = ulid.New(ms, tm.kidEntropy); err != nil {
		return uid, fmt.Errorf("could not generate key id: %w", err)
	}
	return uid, nil
}
