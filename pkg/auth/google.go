package auth

import (
	"context"
	"fmt"

	"github.com/golang-jwt/jwt/v4"
	"google.golang.org/api/idtoken"
)

func (tm *TokenManager) CheckGoogleIDToken(ctx context.Context, credential string) (claims *Claims, err error) {
	var payload *idtoken.Payload
	if payload, err = tm.gValidator.Validate(ctx, credential, tm.conf.GoogleClientID); err != nil {
		return nil, err
	}

	if hd := payload.Claims["hd"].(string); hd != tm.conf.HDClaim {
		return nil, fmt.Errorf("%s is not an authorized domain", hd)
	}

	claims = &Claims{
		RegisteredClaims: jwt.RegisteredClaims{
			Subject: payload.Subject,
		},
		Name:    payload.Claims["name"].(string),
		Email:   payload.Claims["email"].(string),
		Picture: payload.Claims["picture"].(string),
	}

	if payload.Claims["locale"] != nil {
		claims.Locale = payload.Claims["locale"].(string)
	}

	return claims, nil
}
