package middleware

import (
	"blog-api/internal/response"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

func RequireAdmin() gin.HandlerFunc {
	return func(c *gin.Context) {
		role, ok := GetRole(c)
		if !ok {
			response.Error(c, http.StatusUnauthorized, "invalid or missing authorization token")
			c.Abort()
			return
		}

		if !strings.EqualFold(role, "admin") {
			response.Error(c, http.StatusUnauthorized, "invalid or missing authorization token")
			c.Abort()
			return
		}
		c.Next()
	}
}
