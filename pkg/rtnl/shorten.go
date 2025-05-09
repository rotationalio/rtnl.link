package rtnl

import (
	"bytes"
	"errors"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	"github.com/rotationalio/rtnl.link/pkg/api/v1"
	"github.com/rotationalio/rtnl.link/pkg/base62"
	"github.com/rotationalio/rtnl.link/pkg/rtnl/htmx"
	"github.com/rotationalio/rtnl.link/pkg/short"
	"github.com/rotationalio/rtnl.link/pkg/storage"
	"github.com/rotationalio/rtnl.link/pkg/storage/models"

	"github.com/rs/zerolog/log"
	qrcode "github.com/skip2/go-qrcode"
)

const (
	ContentDisposition = "Content-Disposition"
	ContentType        = "Content-Type"
	ContentLength      = "Content-Length"
	AcceptLength       = "Accept-Length"
	ContentTypePNG     = "image/png"
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

	log.Info().Uint64("id", sid).Msg("short url deleted")

	// Redirect the user if this is an HTMX request
	if c.NegotiateFormat(binding.MIMEJSON, binding.MIMEHTML) == binding.MIMEHTML {
		htmx.Redirect(c, http.StatusFound, "")
		return
	}

	c.JSON(http.StatusOK, &api.Reply{Success: true})
}

func (s *Server) ShortURLQRCode(c *gin.Context) {
	var (
		err error
		sid uint64
		qrc *qrcode.QRCode
		uri string
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

	uri, _ = s.conf.MakeOriginURLs(base62.Encode(model.ID))
	if qrc, err = qrcode.New(uri, qrcode.Medium); err != nil {
		log.Warn().Err(err).Uint64("id", sid).Msg("could not create qr code from short url")
		c.JSON(http.StatusInternalServerError, api.ErrorResponse("unable to complete request"))
		return
	}

	// Generate the data buffer
	buf := new(bytes.Buffer)
	if err = qrc.Write(512, buf); err != nil {
		log.Error().Err(err).Msg("could not write png qrcode to download")
		c.JSON(http.StatusInternalServerError, api.ErrorResponse("unable to complete request"))
	}

	// Otherwise set disposition to download to download the file.
	filename := base62.Encode(sid) + ".png"

	// Execute the download request
	c.Header(ContentDisposition, "attachment; filename="+filename)
	c.Header(ContentLength, strconv.Itoa(len(buf.Bytes())))
	c.Header(ContentType, ContentTypePNG)
	c.Writer.WriteHeader(http.StatusOK)
	c.Writer.Write(buf.Bytes())
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
			Target: url.URL,
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
