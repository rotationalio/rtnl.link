package rtnl

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
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

	// By default we attempt to create the model, but if it already exists then we do
	// not modify it and return a 200 status instead of a 201 status.
	code := http.StatusCreated
	if err = s.db.Save(model); err != nil {
		// If the URL already exists in the database return it without an error.
		// If the error is not an already exists error than return 500.
		if !errors.Is(err, storage.ErrAlreadyExists) {
			log.Error().Err(err).Msg("could not store shortened url")
			c.JSON(http.StatusInternalServerError, api.ErrorResponse("unable to complete request"))
			return
		}

		// Attempt to load the already created model from the database.
		if model, err = s.db.LoadInfo(model.ID); err != nil {
			log.Error().Err(err).Msg("could not fetch short url after already exists error")
			c.JSON(http.StatusInternalServerError, api.ErrorResponse("unable to complete request"))
			return
		}

		// If we loaded the model without creating it, then return a 200
		code = http.StatusOK
	}

	// Create the output response to send back to the user.
	out := model.ToAPI()
	out.URL, out.AltURL = s.conf.MakeOriginURLs(sid)

	// Send the shortened event to ensign
	s.analytics.Shortened(out)

	c.Negotiate(code, gin.Negotiate{
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

	// Create the API response to send back to the user.
	out := model.ToAPI()
	out.URL, out.AltURL = s.conf.MakeOriginURLs(base62.Encode(sid))

	c.Negotiate(http.StatusOK, gin.Negotiate{
		Offered:  []string{gin.MIMEHTML, gin.MIMEJSON},
		HTMLName: "links_detail.html",
		HTMLData: out.WebData(),
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

	// Send the deleted event to ensign
	s.analytics.Deleted(c.Param("id"))

	log.Info().Uint64("id", sid).Msg("short url deleted")
	c.JSON(http.StatusOK, &api.Reply{Success: true})
}

func (s *Server) ShortURLList(c *gin.Context) {
	var (
		err  error
		page *api.PageQuery
		out  *api.ShortURLList
	)

	// Bind and validate the page query request
	page = &api.PageQuery{}
	if err = c.BindQuery(page); err != nil {
		log.Warn().Err(err).Msg("could not bind page query")
		c.JSON(http.StatusBadRequest, api.ErrorResponse("could not parse page query from request"))
		return
	}

	if err = page.Validate(); err != nil {
		c.JSON(http.StatusBadRequest, api.ErrorResponse(err))
		return
	}

	// Retrieve the page from the database
	// TODO: pass the page query to the listing function
	var urls []*models.ShortURL
	if urls, err = s.db.List(); err != nil {
		log.Warn().Err(err).Msg("could not retrieve short url list from db")
		c.JSON(http.StatusInternalServerError, api.ErrorResponse("could not complete request"))
		return
	}

	// Create the API response to send back to the user.
	out = &api.ShortURLList{
		URLs: make([]*api.ShortURL, 0, len(urls)),
		Page: &api.PageQuery{},
	}

	for _, url := range urls {
		out.URLs = append(out.URLs, &api.ShortURL{
			URL:    base62.Encode(url.ID),
			Title:  url.Title,
			Visits: url.Visits,
		})
	}

	c.Negotiate(http.StatusOK, gin.Negotiate{
		Offered:  []string{gin.MIMEHTML, gin.MIMEJSON},
		HTMLName: "links_list.html",
		HTMLData: out.WebData(),
		JSONData: out,
	})
}
