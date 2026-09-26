package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// CORS returns a middleware that handles Cross-Origin Resource Sharing (CORS)
// allowing frontend web applications to interact with the API securely.
func CORS() gin.HandlerFunc {
	return func(c *gin.Context) {
		origin := c.GetHeader("Origin")

		if origin != "" {
			c.Header("Access-Control-Allow-Origin", origin)
			c.Header("Access-Control-Allow-Credentials", "true")
		} else {
			c.Header("Access-Control-Allow-Origin", "*")
		}

		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS, HEAD")

		reqHeaders := c.GetHeader("Access-Control-Request-Headers")
		if reqHeaders != "" {
			c.Header("Access-Control-Allow-Headers", reqHeaders)
		} else {
			c.Header("Access-Control-Allow-Headers", "Origin, Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, Accept, Cache-Control, X-Requested-With")
		}

		c.Header("Access-Control-Expose-Headers", "Content-Length, Content-Type, Authorization")
		c.Header("Access-Control-Max-Age", "43200") // 12 hours

		// Handle browser preflight OPTIONS request immediately
		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		c.Next()
	}
}
