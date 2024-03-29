package auth

import (
	"time"

	"github.com/golang-jwt/jwt/v4"
)

type Claims struct {
	jwt.RegisteredClaims
	Name    string `json:"name,omitempty"`
	Email   string `json:"email,omitempty"`
	Picture string `json:"picture,omitempty"`
	Locale  string `json:"locale,omitempty"`
}

// Used to extract expiration and not before timestamps without having to use public keys
var tsparser = &jwt.Parser{SkipClaimsValidation: true}

func ParseUnverified(tks string) (claims *jwt.RegisteredClaims, err error) {
	claims = &jwt.RegisteredClaims{}
	if _, _, err = tsparser.ParseUnverified(tks, claims); err != nil {
		return nil, err
	}
	return claims, nil
}

func ExpiresAt(tks string) (_ time.Time, err error) {
	var claims *jwt.RegisteredClaims
	if claims, err = ParseUnverified(tks); err != nil {
		return time.Time{}, err
	}
	return claims.ExpiresAt.Time, nil
}

func NotBefore(tks string) (_ time.Time, err error) {
	var claims *jwt.RegisteredClaims
	if claims, err = ParseUnverified(tks); err != nil {
		return time.Time{}, err
	}
	return claims.NotBefore.Time, nil
}
