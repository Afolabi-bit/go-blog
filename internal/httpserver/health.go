package httpserver

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

type HealthPinger interface {
	Ping(ctx context.Context) error
}

// Health godoc
// @Summary      Health check
// @Description  Check service availability, database connectivity, and system time
// @Tags         health
// @Produce      json
// @Success      200  {object}  map[string]any
// @Failure      503  {object}  map[string]any
// @Router       /health [get]
func NewHealthHandler(pinger HealthPinger) gin.HandlerFunc {
	return func(c *gin.Context) {
		dbStatus := "connected"
		if pinger != nil {
			if err := pinger.Ping(c.Request.Context()); err != nil {
				c.JSON(http.StatusServiceUnavailable, gin.H{
					"ok":       false,
					"service":  "go-blog",
					"database": "disconnected",
					"error":    err.Error(),
					"time":     time.Now().UTC(),
				})
				return
			}
		}

		c.JSON(http.StatusOK, gin.H{
			"ok":       true,
			"service":  "go-blog",
			"database": dbStatus,
			"time":     time.Now().UTC(),
		})
	}
}

