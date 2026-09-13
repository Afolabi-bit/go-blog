package response

import (
	"github.com/gin-gonic/gin"
)

// Standard response envelope
type Response struct {
	Status  string `json:"status"`            // "success" or "error"
	Message string `json:"message,omitempty"` // Optional human-readable message
	Data    any    `json:"data,omitempty"`    // Optional payload
}

// Success sends a standard success JSON response
func Success(c *gin.Context, statusCode int, message string, data any) {
	c.JSON(statusCode, Response{
		Status:  "success",
		Message: message,
		Data:    data,
	})
}

// Error sends a standard error JSON response
func Error(c *gin.Context, statusCode int, message string) {
	c.JSON(statusCode, Response{
		Status:  "error",
		Message: message,
	})
}
