package app

import (
	"blog-api/internal/authorrequest"
	"blog-api/internal/config"
	"blog-api/internal/db"
	"blog-api/internal/httpserver"
	"blog-api/internal/posts"
	"blog-api/internal/user"
	"context"
	"fmt"

	"github.com/gin-gonic/gin"
)

type App struct {
	Config   config.Config
	Router   *gin.Engine
	Database *db.Mongo
}

func NewApp(ctx context.Context) (*App, error) {
	cfg, err := config.LoadConfig()

	if err != nil {
		return nil, fmt.Errorf("failed to load config: %w", err)
	}

	database, err := db.Connect(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	if err := db.EnsureIndexes(ctx, database.Database); err != nil {
		return nil, fmt.Errorf("failed to initialize database indexes: %w", err)
	}

	userRepo := user.NewRepo(database.Database)
	userService := user.NewService(userRepo, cfg.JWTSecret)
	userHandler := user.NewHandler(userService)

	postRepo := posts.NewRepo(database.Database)
	postService := posts.NewService(postRepo, userRepo)
	postHandler := posts.NewHandler(postService)

	authorRequestRepo := authorrequest.NewRepo(database.Database)
	authorRequestService := authorrequest.NewService(authorRequestRepo, userRepo)
	authorRequestHandler := authorrequest.NewHandler(authorRequestService)

	engine := httpserver.NewRouter(userHandler, postHandler, authorRequestHandler, database, cfg.JWTSecret)

	return &App{
		Config:   cfg,
		Router:   engine,
		Database: database,
	}, nil
}

func (a *App) Close(ctx context.Context) error {
	return a.Database.Disconnect(ctx)
}
