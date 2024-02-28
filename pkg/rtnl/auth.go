package rtnl

import (
	"errors"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/rotationalio/rtnl.link/pkg/api/v1"
	"github.com/rotationalio/rtnl.link/pkg/auth"
	"github.com/rotationalio/rtnl.link/pkg/keygen"
	"github.com/rotationalio/rtnl.link/pkg/passwd"
	"github.com/rotationalio/rtnl.link/pkg/storage"
	"github.com/rs/zerolog/log"
)

const (
	authorization      = "Authorization"
	contextUserClaims  = "user_claims"
	accessTokenCookie  = "access_token"
	refreshTokenCookie = "refresh_token"
)

// used to extract the access token from the header
var (
	bearer = regexp.MustCompile(`^\s*[Bb]earer\s+([a-zA-Z0-9\-]+)\s*$`)
)

func (s *Server) Authenticate(c *gin.Context) {
	// If an access token cookie is available, authenticate using the JWT token
	if cookie, _ := c.Cookie(accessTokenCookie); cookie != "" {
		if err := s.AuthorizeAccessToken(c); err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, api.ErrorResponse(err))
			return
		}

		// Web authentication successful, process rest of response and return.
		c.Next()
		return
	}

	// Authenticate with API Keys if there is no access token
	token, err := GetBearerToken(c)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusUnauthorized, api.ErrorResponse(err))
		return
	}

	clientID, secret, err := ParseToken(token)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusUnauthorized, api.ErrorResponse(err))
		return
	}

	apikey, err := s.db.Retrieve(clientID)
	if err != nil {
		if !errors.Is(err, storage.ErrNotFound) {
			log.Error().Err(err).Msg("could not retrieve apikey from the database")
		}
		log.Debug().Str("clientID", clientID).Err(err).Msg("could not find client id in the database")
		c.AbortWithStatusJSON(http.StatusUnauthorized, api.ErrorResponse(api.ErrUnauthenticated))
		return
	}

	verified, err := passwd.VerifyDerivedKey(apikey.DerivedKey, secret)
	if err != nil {
		log.Error().Err(err).Msg("could not verify derived key")
		c.AbortWithStatusJSON(http.StatusUnauthorized, api.ErrorResponse(api.ErrUnauthenticated))
		return
	}

	if !verified {
		c.AbortWithStatusJSON(http.StatusUnauthorized, api.ErrorResponse(api.ErrUnauthenticated))
		return
	}

	c.Next()
}

func (s *Server) WebAuthenticate(c *gin.Context) {
	if err := s.AuthorizeAccessToken(c); err != nil {
		c.Redirect(http.StatusTemporaryRedirect, "/login")
		c.Abort()
		return
	}

	c.Next()
}

func (s *Server) AuthorizeAccessToken(c *gin.Context) (err error) {
	var (
		accessToken  string
		refreshToken string
		claims       *auth.Claims
	)

	if accessToken, err = c.Cookie(accessTokenCookie); err != nil || accessToken == "" {
		log.Warn().Err(err).Msg("no access token available")
		return api.ErrUnauthenticated
	}

	if claims, err = s.auth.Verify(accessToken); err != nil {
		// Attempt to reauthenticate
		if refreshToken, err = c.Cookie(refreshTokenCookie); err != nil || refreshToken == "" {
			log.Warn().Err(err).Msg("no refresh token available")
			return api.ErrUnauthenticated
		}

		if _, err = s.auth.Verify(refreshToken); err != nil {
			log.Warn().Err(err).Msg("invalid access and refresh token")
			return api.ErrUnauthenticated
		}

		// Extract claims from access token without validation
		if claims, err = s.auth.Parse(accessToken); err != nil {
			log.Warn().Err(err).Msg("could not extract claims from access token")
			return api.ErrUnauthenticated
		}

		var atks, rtks string
		if atks, rtks, err = s.auth.CreateTokenPair(claims); err != nil {
			log.Warn().Err(err).Msg("could not create access and refresh tokens from claims")
			return api.ErrUnauthenticated
		}

		// Set new authentication cookies on refresh
		s.SetAuthCookies(c, atks, rtks)
		return err
	}

	// Add claims to context for use in downstream processing and continue handlers
	c.Set(contextUserClaims, claims)
	return nil
}

func GetBearerToken(c *gin.Context) (tks string, err error) {
	// Attempt to get the access token from the header.
	if header := c.GetHeader(authorization); header != "" {
		match := bearer.FindStringSubmatch(header)
		if len(match) == 2 {
			return match[1], nil
		}
		return "", api.ErrParseBearer
	}

	return "", api.ErrNoAuthorization
}

func ParseToken(token string) (clientID, secret string, err error) {
	parts := strings.Split(token, "-")
	if len(parts) != 2 {
		return "", "", api.ErrInvalidToken
	}

	clientID, secret = parts[0], parts[1]
	if len(clientID) != keygen.KeyIDLength || len(secret) != keygen.SecretLength {
		return "", "", api.ErrInvalidToken
	}

	return clientID, secret, nil
}

func (s *Server) SetAuthCookies(c *gin.Context, accessToken, refreshToken string) (err error) {
	// Figure out the max age for access and refresh tokens from the refresh token
	var expiration time.Time
	if expiration, err = auth.ExpiresAt(refreshToken); err != nil {
		return err
	}

	// Compute the max age of the cookies based on the refresh token expiration.
	maxAge := int(time.Until(expiration).Seconds())

	// If the cookie domain is localhost, then set secure to false for development
	secure := s.conf.Auth.CookieDomain != "localhost"

	c.SetCookie(accessTokenCookie, accessToken, maxAge, "/", s.conf.Auth.CookieDomain, secure, true)
	c.SetCookie(refreshTokenCookie, refreshToken, maxAge, "/", s.conf.Auth.CookieDomain, secure, true)
	return nil
}

func (s *Server) ClearAuthCookies(c *gin.Context) {
	// If the cookie domain is localhost, then set secure to false for development
	secure := s.conf.Auth.CookieDomain != "localhost"

	// Remove authentication cookies by setting expired empty string cookies
	c.SetCookie(accessTokenCookie, "", -1, "/", s.conf.Auth.CookieDomain, secure, true)
	c.SetCookie(refreshTokenCookie, "", -1, "/", s.conf.Auth.CookieDomain, secure, true)
}
