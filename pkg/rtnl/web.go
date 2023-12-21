package rtnl

import (
	"net/http"

	"github.com/gin-gonic/gin"
	api "github.com/rotationalio/rtnl.link/pkg/api/v1"
)

func (s *Server) Index(c *gin.Context) {
	data := api.NewWebData()
	c.HTML(http.StatusOK, "index.html", data)
}

func (s *Server) List(c *gin.Context) {
	data := api.NewWebData()
	c.HTML(http.StatusOK, "list.html", data)
}
