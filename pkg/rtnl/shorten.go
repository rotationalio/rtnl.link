package rtnl

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/rotationalio/rtnl.link/pkg"
	"github.com/rotationalio/rtnl.link/pkg/api/v1"
	"github.com/rotationalio/rtnl.link/pkg/base62"
	"github.com/rotationalio/rtnl.link/pkg/short"
	"github.com/rotationalio/rtnl.link/pkg/storage"
	"github.com/rotationalio/rtnl.link/pkg/storage/models"
	"github.com/rs/zerolog/log"
)

func (s *Server) ShortenURL(c *gin.Context) {
	var (
		err  error
		sid  string
		long *api.LongURL
	)

	if err = c.Bind(&long); err != nil {
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
		c.JSON(http.StatusInternalServerError, api.ErrorResponse("unable to complete request"))
		return
	}

	// Save URL to the database
	model := &models.ShortURL{URL: long.URL}
	model.ID, _ = base62.Decode(sid)
	model.Expires, _ = long.ExpiresAt()

	if err = s.db.Save(model); err != nil {
		if errors.Is(err, storage.ErrAlreadyExists) {
			c.JSON(http.StatusConflict, api.ErrorResponse("shortened url already exists"))
			return
		}

		log.Error().Err(err).Msg("could not store shortened url")
		c.JSON(http.StatusInternalServerError, api.ErrorResponse("unable to complete request"))
		return
	}

	out := &api.ShortURL{}
	out.URL, out.AltURL = s.conf.MakeOriginURLs(sid)
	if !model.Expires.IsZero() {
		out.Expires = &model.Expires
	}

	c.Negotiate(http.StatusCreated, gin.Negotiate{
		Offered:  []string{gin.MIMEHTML, gin.MIMEJSON},
		HTMLName: "created.html",
		HTMLData: out,
		JSONData: out,
	})
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

	// Lookup URL info from the database
	var model *models.ShortURL
	if model, err = s.db.LoadInfo(sid); err != nil {
		if errors.Is(err, storage.ErrNotFound) {
			c.JSON(http.StatusNotFound, api.ErrorResponse("short url not found"))
			return
		}

		log.Warn().Err(err).Uint64("id", sid).Msg("could not load url from database")
		c.JSON(http.StatusInternalServerError, api.ErrorResponse("unable to complete request"))
		return
	}

	out := &api.ShortURL{
		Title:       model.Title,
		Description: model.Description,
		Visits:      model.Visits,
		CampaignID:  model.CampaignID,
		Campaigns:   model.Campaigns,
	}
	out.URL, out.AltURL = s.conf.MakeOriginURLs(base62.Encode(sid))

	if !model.Expires.IsZero() {
		out.Expires = &model.Expires
	}

	if !model.Created.IsZero() {
		out.Created = &model.Created
	}

	if !model.Modified.IsZero() {
		out.Modified = &model.Modified
	}

	c.Negotiate(http.StatusOK, gin.Negotiate{
		Offered:  []string{gin.MIMEHTML, gin.MIMEJSON},
		HTMLName: "info.html",
		HTMLData: &InfoDetail{WebData: WebData{Version: pkg.Version()}, Info: out},
		JSONData: out,
	})
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

	// Delete URL info from the database
	if err = s.db.Delete(sid); err != nil {
		if errors.Is(err, storage.ErrNotFound) {
			c.JSON(http.StatusNotFound, api.ErrorResponse("short url not found"))
			return
		}

		log.Warn().Err(err).Uint64("id", sid).Msg("could not delete url from database")
		c.JSON(http.StatusInternalServerError, api.ErrorResponse("unable to complete request"))
		return
	}

	log.Info().Uint64("id", sid).Msg("short url deleted")
	c.JSON(http.StatusOK, &api.Reply{Success: true})
}
