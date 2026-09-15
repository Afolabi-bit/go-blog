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
			response.Error(c, http.StatusUnauthorized, "you don't have the permission to perform this action.")
			c.Abort()
			return
		}

		if !strings.EqualFold(role, "admin") {
			response.Error(c, http.StatusForbidden, "you don't have the permission to perform this action.")
			c.Abort()
			return
		}
		c.Next()
	}
}

func RequireRoles(allowedRoles ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		role, ok := GetRole(c)
		if !ok {
			response.Error(c, http.StatusUnauthorized, "you don't have the permission to perform this action.")
			c.Abort()
			return
		}

		for _, r := range allowedRoles {
			if strings.EqualFold(role, r) {
				c.Next()
				return
			}
		}
		response.Error(c, http.StatusForbidden, "you don't have the permission to perform this action.")
		c.Abort()
	}
}
