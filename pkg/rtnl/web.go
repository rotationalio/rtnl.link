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
	"github.com/rs/zerolog/log"
)

// Index returns the home page and landing dashboard.
func (s *Server) Index(c *gin.Context) {
	data := api.GetWebData()
	c.HTML(http.StatusOK, "index.html", data)
}

func (s *Server) List(c *gin.Context) {
	data := api.GetWebData()
	c.HTML(http.StatusOK, "list.html", data)
}

// ShortURL detail returns the detail page for a short URL including analytics info.
func (s *Server) ShortURLDetail(c *gin.Context) {
	// Get URL parameter from input
	data := gin.H{
		"ID":      c.Param("id"),
		"Version": pkg.Version(),
	}
	c.HTML(http.StatusOK, "info.html", data)
}

// Login page returns the web-based login for a Google sign-in button.
func (s *Server) LoginPage(c *gin.Context) {
	data := api.GetLoginData()
	c.HTML(http.StatusOK, "login.html", data)
}

// Login handles the POST request from Google when a user successfully logs in.
func (s *Server) Login(c *gin.Context) {
	// TODO: switch to cookie-based authentication!
	var (
		err error
		in  *api.LoginForm
	)

	in = &api.LoginForm{}
	if err = c.Bind(in); err != nil {
		log.Warn().Err(err).Msg("could not bind form input")
		c.JSON(http.StatusInternalServerError, api.ErrorResponse(err))
		return
	}

	if in.Credential == "" {
		c.JSON(http.StatusBadRequest, api.ErrorResponse("jwt credential is required"))
		return
	}

	// Parse the JWT id token from Google and validate it
	if err = s.ValidateGoogleJWT(in.Credential); err != nil {
		c.JSON(http.StatusUnauthorized, api.ErrorResponse(err))
		return
	}

	// Store the access token as a cookie on the outgoing response then redirect the
	// user back to the home page or to the next page if it has been provided.

	c.JSON(http.StatusOK, gin.H{"success": true, "credential": in.Credential})
}

// Updates serves a web socket connection to stream live updates back to the client.
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
