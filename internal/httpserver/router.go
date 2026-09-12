package httpserver

import "github.com/gin-gonic/gin"

func NewRouter() *gin.Engine {
	router := gin.Default()

	router.GET("/health", health)

	return router
}
