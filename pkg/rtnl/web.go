package rtnl

import (
	"errors"
	"io"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/rotationalio/rtnl.link/pkg"
	api "github.com/rotationalio/rtnl.link/pkg/api/v1"
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

	// TODO: subscribe to ensign here to push notifications down
	// In the meantime, just write data back to the server
	i := 0
	for {
		i++
		message := &api.Click{
			Time:      time.Now().Truncate(time.Hour).Format("2006-01-02 15:00"),
			Views:     1,
			UserAgent: "Chrome",
			IPAddr:    "10.10.27.1",
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
