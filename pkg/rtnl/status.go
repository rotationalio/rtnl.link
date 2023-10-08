package rtnl

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/rotationalio/rtnl.link/pkg"
	"github.com/rotationalio/rtnl.link/pkg/api/v1"
)

const (
	serverStatusOk          = "ok"
	serverStatusStopping    = "stopping"
	serverStatusMaintenance = "maintenance"
)

// Available is middleware that uses healthy boolean to return a service unavailable
// http status code if the server is shutting down or in maintenance mode. This
// middleware must be fairly early on in the chain to ensure that complex handling does
// not slow the shutdown of the server.
func (s *Server) Available() gin.HandlerFunc {
	// The server starts in maintenance mode and doesn't change during runtime, so
	// determine what the unhealthy status string is going to be prior to the closure.
	status := serverStatusStopping
	if s.conf.Maintenance {
		status = serverStatusMaintenance
	}

	return func(c *gin.Context) {
		// Check the health status
		if s.conf.Maintenance || !s.IsHealthy() {
			out := api.StatusReply{
				Status:  status,
				Uptime:  s.Uptime().String(),
				Version: pkg.Version(),
			}

			// Write the 503 response
			c.JSON(http.StatusServiceUnavailable, out)

			// Stop processing the request if the server is not healthy
			c.Abort()
			return
		}

		// Continue processing the request
		c.Next()
	}
}

func (s *Server) Status(c *gin.Context) {
	c.JSON(http.StatusOK, api.StatusReply{
		Status:  serverStatusOk,
		Uptime:  s.Uptime().String(),
		Version: pkg.Version(),
	})
}

func (s *Server) NotFound(c *gin.Context) {
	c.JSON(http.StatusNotFound, api.ErrNotFoundReply)
}

func (s *Server) NotAllowed(c *gin.Context) {
	c.JSON(http.StatusMethodNotAllowed, api.ErrNotAllowedReply)
}
