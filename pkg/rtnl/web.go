package rtnl

import (
	"errors"
	"io"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/rotationalio/go-ensign"
	"github.com/rotationalio/rtnl.link/pkg"
	api "github.com/rotationalio/rtnl.link/pkg/api/v1"
	"github.com/rotationalio/rtnl.link/pkg/passwd"
	"github.com/rotationalio/rtnl.link/pkg/storage"
	"github.com/rotationalio/rtnl.link/pkg/storage/models"
	"github.com/rs/zerolog/log"
)

func (s *Server) Index(c *gin.Context) {
	data := api.NewWebData()
	c.HTML(http.StatusOK, "index.html", data)
}

func (s *Server) List(c *gin.Context) {
	data := api.NewWebData()
	c.HTML(http.StatusOK, "list.html", data)
}

func (s *Server) ShortURLDetail(c *gin.Context) {
	// Get URL parameter from input
	data := gin.H{
		"ID":      c.Param("id"),
		"Version": pkg.Version(),
	}
	c.HTML(http.StatusOK, "info.html", data)
}

func (s *Server) LoginPage(c *gin.Context) {
	data := api.NewWebData()
	c.HTML(http.StatusOK, "login.html", data)
}

func (s *Server) Login(c *gin.Context) {
	// TODO: switch to cookie-based authentication!
	var (
		err      error
		in       *api.LoginForm
		clientID string
		secret   string
		apikey   *models.APIKey
		verified bool
	)

	in = &api.LoginForm{}
	if err = c.Bind(in); err != nil {
		log.Warn().Err(err).Msg("could not bind form input")
		c.JSON(http.StatusInternalServerError, api.ErrorResponse(err))
		return
	}

	if in.APIKey == "" {
		c.JSON(http.StatusBadRequest, api.ErrorResponse("apikey is required"))
		return
	}

	// Parse the token and validate it
	if clientID, secret, err = ParseToken(in.APIKey); err != nil {
		c.JSON(http.StatusBadRequest, api.ErrorResponse("invalid api key"))
		return
	}

	if apikey, err = s.db.Retrieve(clientID); err != nil {
		if !errors.Is(err, storage.ErrNotFound) {
			log.Error().Err(err).Msg("could not retrieve apikey from the database")
		}

		c.JSON(http.StatusBadRequest, api.ErrorResponse("invalid api key"))
		return
	}

	if verified, err = passwd.VerifyDerivedKey(apikey.DerivedKey, secret); err != nil {
		log.Error().Err(err).Msg("could not verify derived key")
		c.JSON(http.StatusInternalServerError, api.ErrorResponse(err))
		return
	}

	if !verified {
		c.JSON(http.StatusBadRequest, api.ErrorResponse("invalid api key"))
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "apikey": in.APIKey})
}

func (s *Server) Updates(c *gin.Context) {
	var (
		err    error
		conn   *websocket.Conn
		linkID string
		sub    *ensign.Subscription
	)

	// Upgrade the connection to an http/2 connection for websockets
	if conn, err = s.upgrader.Upgrade(c.Writer, c.Request, nil); err != nil {
		log.Error().Err(err).Msg("could not upgrade to websocket connection")
		c.JSON(http.StatusInternalServerError, api.ErrorResponse(err))
		return
	}
	defer conn.Close()

	// Parse the URL if given for filtering (or not) the Ensign stream
	linkID = c.Param("id")
	log.Info().Str("link_id", linkID).Msg("updates websocket opened")

	// Subscribe to the ensign topic for updates
	if sub, err = s.analytics.Subscribe(); err != nil {
		log.Error().Err(err).Msg("could not connect to ensign")
		c.JSON(http.StatusInternalServerError, api.ErrorResponse(err))
		return
	}
	defer sub.Close()

	// TODO: write some initial data to the websocket to display the graph.
	if err = conn.WriteJSON(api.Clicked(c)); err != nil {
		log.Error().Err(err).Msg("could not send message to establish connection")
		c.JSON(http.StatusInternalServerError, api.ErrorResponse(err))
		return
	}

	// In the meantime, just write data back to the server
	for event := range sub.C {
		log.Debug().Msg("waiting for updates")

		// Only publish click events to the updates stream
		if event.Type.Name != "Click" {
			continue
		}

		// Filter the message if necessary
		if linkID != "" && event.Metadata["id"] != linkID {
			continue
		}

		message := &api.Click{}
		if err = message.UnmarshalValue(event.Data); err != nil {
			log.Warn().Err(err).Str("type", event.Type.String()).Msg("could not unmarshal click event for update stream")
			continue
		}

		if err = conn.WriteJSON(message); err != nil {
			if !errors.Is(err, io.EOF) {
				log.Error().Err(err).Msg("could not write message to websocket")
				return
			}

			log.Info().Msg("web sockets closed")
			return
		}

		time.Sleep(time.Second)
	}
}
