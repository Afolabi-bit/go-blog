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

	// auth
	authGroup := router.Group("/auth")

	{
		authGroup.POST("/register", userHandler.Register)
		authGroup.POST("/login", userHandler.Login)
	}

	// user
	userGroup := router.Group("/user")

	userGroup.Use(middleware.AuthRequired(jwtSecret))

	{
		userGroup.GET("/iam", userHandler.Me)
	}

	// posts
	postGroup := router.Group("/api/post")
	{
		postGroup.GET("")
		postGroup.GET("/:id", postHandler.GetPostByID)
	}

	// author
	authorGroup := router.Group("/api")
	authorGroup.Use(middleware.AuthRequired(jwtSecret))
	authorGroup.Use(middleware.RequireRoles(user.RoleAuthor, user.RoleAdmin))

	{
		authorGroup.POST("/posts", postHandler.CreatePost)
		authorGroup.GET("/my-posts", postHandler.ListMyPosts)
		authorGroup.PATCH("/my-posts", postHandler.UpdatePost)
		authorGroup.DELETE("/my-posts", postHandler.DeletePost)

	}

	// admin
	adminGroup := router.Group("/admin")
	adminGroup.Use(middleware.AuthRequired(jwtSecret))
	adminGroup.Use(middleware.RequireAdmin())

	{
		adminGroup.GET("/posts", postHandler.ListAllAdmin)
		adminGroup.DELETE("/posts/:id", postHandler.DeletePost)
	}
	return router
}
