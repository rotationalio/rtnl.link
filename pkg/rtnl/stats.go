package rtnl

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/rotationalio/rtnl.link/pkg/api/v1"
	"github.com/rotationalio/rtnl.link/pkg/storage/models"
	"github.com/rs/zerolog/log"
)

func (s *Server) ShortcrustStats(c *gin.Context) {
	var (
		err    error
		counts *models.Counts
		out    *api.ShortcrustInfo
	)

	if counts, err = s.db.Counts(); err != nil {
		log.Warn().Err(err).Msg("could not count objects in database")
		c.JSON(http.StatusInternalServerError, api.ErrorResponse("unable to complete request"))
		return
	}

	out = counts.ToAPI()
	c.Negotiate(http.StatusOK, gin.Negotiate{
		Offered:  []string{gin.MIMEHTML, gin.MIMEJSON},
		HTMLName: "stats.html",
		HTMLData: out.WebData(),
		JSONData: out,
	})
}
