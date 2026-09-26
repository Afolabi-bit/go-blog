package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"blog-api/internal/app"
)

// @title           Blog REST API
// @version         1.0
// @description     A multi-tenant Blog API with JWT authentication and Role-Based Access Control (RBAC).
// @termsOfService  http://swagger.io/terms/

// @contact.name   API Support
// @contact.email  support@blog.com

// @host      localhost:5000
// @BasePath  /
// @schemes   http https

// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
// @description Type "Bearer" followed by a space and your JWT token. Example: "Bearer eyJhbGciOi..."

func main() {
	ctx := context.Background()

	app, err := app.NewApp(ctx)
	if err != nil {
		log.Fatalf("Failed to start application: %v", err)
	}

	serve := http.Server{
		Addr:              fmt.Sprintf(":%s", app.Config.Port),
		Handler:           app.Router,
		ReadHeaderTimeout: time.Second * 5,
		IdleTimeout:       time.Second * 60,
	}

	go func() {
		log.Printf("Server listening on port %s", app.Config.Port)
		err := serve.ListenAndServe()

		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("Unexpected server shutdown: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	sig := <-quit
	log.Printf("Received signal '%v'. Shutting down server...\n", sig)

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()

	if err := serve.Shutdown(shutdownCtx); err != nil {
		log.Printf("Server forced to shutdown: %v", err)
	} else {
		log.Println("HTTP server stopped gracefully")
	}

	dbCtx, dbCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer dbCancel()

	if err := app.Close(dbCtx); err != nil {
		log.Printf("Error closing database connection: %v", err)
	} else {
		log.Println("Database connection closed cleanly")
	}

	log.Println("Server exiting")
}
