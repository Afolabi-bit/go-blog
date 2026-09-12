package main

import (
	"log"

	"blog-api/internal/config"
	"blog-api/internal/httpserver"
)

func main() {
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// ctx := context.Background()

	// mongo, err := db.Connect(ctx, cfg)
	// if err != nil {
	// 	log.Fatalf("Failed to connect to database: %v", err)
	// }

	// defer mongo.Disconnect(ctx)

	router := httpserver.NewRouter()

	if err := router.Run(":" + cfg.Port); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
