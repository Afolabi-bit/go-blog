package httpserver

import (
	_ "blog-api/docs"
	"blog-api/internal/authorrequest"
	"blog-api/internal/middleware"
	"blog-api/internal/posts"
	"blog-api/internal/user"

	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

func NewRouter(
	userHandler *user.Handler,
	postHandler *posts.Handler,
	authorRequestHandler *authorrequest.Handler,
	pinger HealthPinger,
	jwtSecret string,
) *gin.Engine {
	router := gin.New()
	router.HandleMethodNotAllowed = true
	router.Use(middleware.CORS())
	router.Use(gin.Logger())
	router.Use(gin.Recovery())

	router.GET("/health", NewHealthHandler(pinger))
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
		userGroup.POST("/author-request", authorRequestHandler.Submit)
		userGroup.GET("/author-request", authorRequestHandler.GetMyRequest)
	}

	// posts
	postGroup := router.Group("/api/posts")
	postGroup.Use(middleware.AuthOptional(jwtSecret))
	{
		postGroup.GET("", postHandler.ListPublicPosts)
		postGroup.GET("/slug/:slug", postHandler.GetPostBySlug)
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
		adminGroup.GET("/author-requests", authorRequestHandler.List)
		adminGroup.PATCH("/author-requests/:id/review", authorRequestHandler.Review)
	}
	return router
}
