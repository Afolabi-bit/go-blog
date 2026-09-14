package main

import (
	"context"
	"log"

	"blog-api/internal/config"
	"blog-api/internal/db"
	"blog-api/internal/httpserver"
	"blog-api/internal/user"
)

func main() {
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	ctx := context.Background()

	mongo, err := db.Connect(ctx, cfg)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}

	defer mongo.Disconnect(ctx)

	userRepo := user.NewRepo(mongo.Database)
	userService := user.NewService(userRepo, cfg.JWTSecret)
	userHandler := user.NewHandler(userService)

	router := httpserver.NewRouter(userHandler, cfg.JWTSecret)

	if err := router.Run(":" + cfg.Port); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
