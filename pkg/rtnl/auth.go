package rtnl

import (
	"errors"
	"net/http"
	"regexp"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/rotationalio/rtnl.link/pkg/api/v1"
	"github.com/rotationalio/rtnl.link/pkg/keygen"
	"github.com/rotationalio/rtnl.link/pkg/passwd"
	"github.com/rotationalio/rtnl.link/pkg/storage"
	"github.com/rs/zerolog/log"
)

const (
	authorization = "Authorization"
)

// used to extract the access token from the header
var (
	bearer = regexp.MustCompile(`^\s*[Bb]earer\s+([a-zA-Z0-9\-]+)\s*$`)
)

func (s *Server) Authenticate(c *gin.Context) {
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
