package httpserver

import (
	"blog-api/internal/middleware"
	"blog-api/internal/posts"
	"blog-api/internal/user"

	"github.com/gin-gonic/gin"
)

func NewRouter(userHandler *user.Handler, postHandler *posts.Handler, jwtSecret string) *gin.Engine {
	router := gin.New()
	router.HandleMethodNotAllowed = true
	router.Use(gin.Logger())
	router.Use(gin.Recovery())

	router.GET("/health", health)

	authGroup := router.Group("/auth")

	{
		authGroup.POST("/register", userHandler.Register)
		authGroup.POST("/login", userHandler.Login)
	}

	userGroup := router.Group("/user")

	userGroup.Use(middleware.AuthRequired(jwtSecret))

	{
		userGroup.GET("/iam", userHandler.Me)
	}
	return router
}
