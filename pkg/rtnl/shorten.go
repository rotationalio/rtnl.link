package rtnl

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/rotationalio/rtnl.link/pkg/api/v1"
	"github.com/rotationalio/rtnl.link/pkg/base62"
	"github.com/rotationalio/rtnl.link/pkg/short"
	"github.com/rs/zerolog/log"
)

func (s *Server) ShortenURL(c *gin.Context) {
	var (
		err  error
		sid  string
		long *api.LongURL
	)

	if err = c.BindJSON(&long); err != nil {
		log.Warn().Err(err).Msg("could not parse shorten url request")
		c.JSON(http.StatusBadRequest, api.ErrUnparsable)
		return
	}

	if err = long.Validate(); err != nil {
		c.Error(err)
		c.JSON(http.StatusBadRequest, api.ErrorResponse(err))
		return
	}

	// Generate the short URL id from a hash of the input URL
	if sid, err = short.URL(long.URL); err != nil {
		c.Error(err)
		c.JSON(http.StatusInternalServerError, "unable to complete request")
		return
	}

	// TODO: save URL to the database
	// TODO: do not hardcode URIs, but fetch from config
	out := &api.ShortURL{
		URL:    "https://rtnl.link/" + sid,
		AltURL: "https://r8l.co/" + sid,
	}

	out.Expires, _ = long.ExpiresAt()
	c.JSON(http.StatusCreated, out)
}

func (s *Server) ShortURLInfo(c *gin.Context) {
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

	// TODO: lookup URL info from the database
	// TODO: perform expiration check
	surl := base62.Encode(sid)
	out := &api.ShortURL{
		URL:    "https://rtnl.link/" + surl,
		AltURL: "https://r8l.co/" + surl,
	}
	c.JSON(http.StatusOK, out)
}

func (s *Server) DeleteShortURL(c *gin.Context) {
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

	// TODO: delete URL info from the database
	log.Info().Uint64("id", sid).Msg("short url deleted")
	c.JSON(http.StatusOK, &api.Reply{Success: true})
}
