package rtnl

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/rotationalio/rtnl.link/pkg"
	api "github.com/rotationalio/rtnl.link/pkg/api/v1"
	"github.com/rotationalio/rtnl.link/pkg/auth"
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
	var claims *auth.Claims
	if claims, err = s.auth.CheckGoogleIDToken(c.Request.Context(), in.Credential); err != nil {
		c.JSON(http.StatusUnauthorized, api.ErrorResponse(err))
		return
	}

	// Create access and refresh tokens from the claims
	var atks, rtks string
	if atks, rtks, err = s.auth.CreateTokenPair(claims); err != nil {
		log.Warn().Err(err).Msg("could not create access and refresh token pair from claims")
		c.JSON(http.StatusInternalServerError, api.ErrorResponse("could not create user credentials"))
		return
	}

	// Store the access token as a cookie on the outgoing response
	if err = s.SetAuthCookies(c, atks, rtks); err != nil {
		log.Warn().Err(err).Msg("could not parse expiration of refresh token")
		c.JSON(http.StatusInternalServerError, api.ErrorResponse("could not create user credentials"))
		return
	}

	// Redirect the user back to the home page
	c.Redirect(http.StatusFound, "/")
}

func (s *Server) Logout(c *gin.Context) {
	// Remove authentication cookies and redirect to the login page
	s.ClearAuthCookies(c)
	c.Redirect(http.StatusFound, "/login")
}

// Updates serves a web socket connection to stream live updates back to the client.
func (s *Server) Updates(c *gin.Context) {
	var (
		err    error
		conn   *websocket.Conn
		linkID string
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

	c.AbortWithStatus(http.StatusNotImplemented)
}
