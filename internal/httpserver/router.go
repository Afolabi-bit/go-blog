package httpserver

import (
	_ "blog-api/docs"
	"blog-api/internal/middleware"
	"blog-api/internal/posts"
	"blog-api/internal/user"

	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

func NewRouter(userHandler *user.Handler, postHandler *posts.Handler, jwtSecret string) *gin.Engine {
	router := gin.New()
	router.HandleMethodNotAllowed = true
	router.Use(gin.Logger())
	router.Use(gin.Recovery())

	router.GET("/health", health)
	router.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	// auth
	authGroup := router.Group("/auth")

	{
		authGroup.POST("/register", userHandler.Register)
		authGroup.POST("/login", userHandler.Login)
		authGroup.POST("/logout", middleware.AuthRequired(jwtSecret), userHandler.Logout)
	}

	// user
	userGroup := router.Group("/user")

	userGroup.Use(middleware.AuthRequired(jwtSecret))

	{
		userGroup.GET("/iam", userHandler.Me)
	}

	// posts
	postGroup := router.Group("/api/posts")
	postGroup.Use(middleware.AuthOptional(jwtSecret))
	{
		postGroup.GET("", postHandler.ListPublicPosts)
		postGroup.GET("/:id", postHandler.GetPostByID)
	}

	// author
	authorGroup := router.Group("/api")
	authorGroup.Use(middleware.AuthRequired(jwtSecret))
	authorGroup.Use(middleware.RequireRoles(user.RoleAuthor, user.RoleAdmin))

	{
		authorGroup.POST("/posts", postHandler.CreatePost)
		authorGroup.GET("/my-posts", postHandler.ListMyPosts)
		authorGroup.PATCH("/posts/:id", postHandler.UpdatePost)
		authorGroup.DELETE("/posts/:id", postHandler.DeletePost)
	}

	// admin
	adminGroup := router.Group("/api/admin")
	adminGroup.Use(middleware.AuthRequired(jwtSecret))
	adminGroup.Use(middleware.RequireAdmin())

	{
		adminGroup.GET("/posts", postHandler.ListAllAdmin)
		adminGroup.DELETE("/posts/:id", postHandler.DeletePost)
	}
	return router
}
