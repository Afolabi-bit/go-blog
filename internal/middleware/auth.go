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
			response.Error(c, http.StatusUnauthorized, auth.ErrMissingAuthHeader.Error())
			c.Abort()
			return
		}

		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || !strings.EqualFold(parts[0], "bearer") {
			response.Error(c, http.StatusUnauthorized, auth.ErrInvalidToken.Error())
			c.Abort()
			return
		}

		tokenStr := strings.TrimSpace(parts[1])
		if tokenStr == "" {
			response.Error(c, http.StatusUnauthorized, auth.ErrInvalidToken.Error())
			c.Abort()
			return
		}

		claims, err := auth.ParseToken(jwtSecret, tokenStr)
		if err != nil {
			response.Error(c, http.StatusUnauthorized, auth.ErrInvalidToken.Error())
			c.Abort()
			return
		}

		c.Set(ctxUserID, claims.Subject)
		c.Set(ctxUserRole, claims.Role)
		c.Next()
	}
}

// AuthOptional extracts user identity from the Authorization header if present and valid,
// but allows unauthenticated / guest requests to proceed.
func AuthOptional(jwtSecret string) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.Next()
			return
		}

		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || !strings.EqualFold(parts[0], "bearer") {
			c.Next()
			return
		}

		tokenStr := strings.TrimSpace(parts[1])
		if tokenStr == "" {
			c.Next()
			return
		}

		claims, err := auth.ParseToken(jwtSecret, tokenStr)
		if err == nil {
			c.Set(ctxUserID, claims.Subject)
			c.Set(ctxUserRole, claims.Role)
		}

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
