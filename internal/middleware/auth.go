package middleware

import (
	"blog-api/internal/auth"
	"blog-api/internal/response"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

const (
	ctxUserID   = "ctxUserID"
	ctxUserRole = "ctxUserRole"
)

func AuthRequired(jwtSecret string) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")

		if authHeader == "" {
			response.Error(c, http.StatusUnauthorized, "invalid or missing authorization token")
			c.Abort()
			return
		}

		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || parts[0] != "Bearer" {
			response.Error(c, http.StatusUnauthorized, "invalid or missing authorization token")
			c.Abort()
			return
		}

		scheme := strings.ToLower(strings.TrimSpace(parts[0]))
		tokenStr := parts[1]

		if scheme != "bearer" {
			response.Error(c, http.StatusUnauthorized, "invalid or missing authorization token")
			c.Abort()
			return
		}

		if tokenStr == "" {
			response.Error(c, http.StatusUnauthorized, "invalid or missing authorization token")
			c.Abort()
			return
		}

		claims, err := auth.ParseToken(jwtSecret, tokenStr)
		if err != nil {
			response.Error(c, http.StatusUnauthorized, "invalid or missing authorization token")
			c.Abort()
			return
		}

		c.Set(ctxUserID, claims.Subject)
		c.Set(ctxUserRole, claims.Role)
		c.Next()
	}
}

func GetUserID(c *gin.Context) (string, bool) {
	val, ok := c.Get(ctxUserID)
	if !ok {
		return "", false
	}

	userId, ok := val.(string)
	return userId, ok
}

func GetRole(c *gin.Context) (string, bool) {
	val, ok := c.Get(ctxUserRole)
	if !ok {
		return "", false
	}

	role, ok := val.(string)
	return role, ok
}
