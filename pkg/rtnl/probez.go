package rtnl

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// Determines if the server is healthy or not.
func (s *Server) IsHealthy() bool {
	s.RLock()
	defer s.RUnlock()
	return s.healthy
}

// Determines if the server is ready or not.
func (s *Server) IsReady() bool {
	s.RLock()
	defer s.RUnlock()
	return s.ready
}

// Set the server health state to the status bool.
func (s *Server) SetHealthy(status bool) {
	s.Lock()
	defer s.Unlock()
	s.healthy = status
}

// Set the server ready state to the status bool.
func (s *Server) SetReady(status bool) {
	s.Lock()
	defer s.Unlock()
	s.ready = status
}

func (s *Server) Healthz(c *gin.Context) {
	status := http.StatusOK
	if !s.IsHealthy() {
		status = http.StatusServiceUnavailable
	}
	c.Data(status, "text/plain", []byte(http.StatusText(status)))
}

func (s *Server) Readyz(c *gin.Context) {
	status := http.StatusOK
	if !s.IsReady() {
		status = http.StatusServiceUnavailable
	}
	c.Data(status, "text/plain", []byte(http.StatusText(status)))
}
