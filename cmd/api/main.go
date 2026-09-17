package main

import (
	"context"
	"log"

	"blog-api/internal/app"
	"blog-api/internal/config"
)

func main() {
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	ctx := context.Background()

	app, err := app.NewApp(ctx)

	if err != nil {
		log.Fatalf("Failed to start application: %v", err)
	}

	if err := app.Router.Run(":" + cfg.Port); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
