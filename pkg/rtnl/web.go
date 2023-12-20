package rtnl

import (
	"embed"
	"net/http"

	"github.com/gin-gonic/gin"
)

//go:embed all:templates
//go:embed all:static
var content embed.FS

func (s *Server) Index(c *gin.Context) {
	c.HTML(http.StatusOK, "index.html", nil)
}
