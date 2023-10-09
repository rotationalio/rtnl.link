package rtnl

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/rotationalio/rtnl.link/pkg/api/v1"
	"github.com/rotationalio/rtnl.link/pkg/base62"
	"github.com/rs/zerolog/log"
)

func (s *Server) Redirect(c *gin.Context) {
	var (
		err error
		sid uint64
	)

	// Get URL parameter from input
	if sid, err = base62.Decode(c.Param("id")); err != nil {
		log.Debug().Err(err).Str("input", c.Param("id")).Msg("could not parse user input")
		c.JSON(http.StatusNotFound, api.ErrNotFoundReply)
		return
	}

	// TODO: fetch long url from the database
	// TODO: increment visits in the database
	// TODO: check expiration for the URL
	log.Info().Uint64("id", sid).Msg("redirecting user")
	c.Redirect(http.StatusMovedPermanently, "https://rotational.io")
}
