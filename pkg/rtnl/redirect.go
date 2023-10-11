package rtnl

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/rotationalio/rtnl.link/pkg/api/v1"
	"github.com/rotationalio/rtnl.link/pkg/base62"
	"github.com/rotationalio/rtnl.link/pkg/storage"
	"github.com/rs/zerolog/log"
)

func (s *Server) Redirect(c *gin.Context) {
	var (
		err error
		sid uint64
		url string
	)

	// Get URL parameter from input
	if sid, err = base62.Decode(c.Param("id")); err != nil {
		log.Debug().Err(err).Str("input", c.Param("id")).Msg("could not parse user input")
		c.JSON(http.StatusNotFound, api.ErrNotFoundReply)
		return
	}

	if url, err = s.db.Load(sid); err != nil {
		if errors.Is(err, storage.ErrNotFound) {
			c.JSON(http.StatusNotFound, "short url not found")
			return
		}

		log.Warn().Err(err).Uint64("id", sid).Msg("could not retrieve short url from database")
		c.JSON(http.StatusInternalServerError, api.ErrorResponse("could not process request"))
		return
	}

	log.Info().Uint64("id", sid).Str("url", url).Msg("redirecting user")
	c.Redirect(http.StatusFound, url)
}
