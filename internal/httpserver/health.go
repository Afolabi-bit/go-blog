package httpserver

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

// Health godoc
// @Summary      Health check
// @Description  Check service availability and system time
// @Tags         health
// @Produce      json
// @Success      200  {object}  map[string]any
// @Router       /health [get]
func health(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"ok":      true,
		"service": "go-blog",
		"time":    time.Now().UTC(),
	})
}
