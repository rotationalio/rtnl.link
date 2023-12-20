package rtnl

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func (s *Server) Index(c *gin.Context) {
	data := NewWebData()
	c.HTML(http.StatusOK, "index.html", data)
}
