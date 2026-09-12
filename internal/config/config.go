package config

import (
	"fmt"
	"os"
	"strings"

	"github.com/joho/godotenv"
)

type Config struct {
	Port          string
	MongoURI      string
	MongoDatabase string
	JWTSecret     string
}

func LoadConfig() (Config, error) {
	_ = godotenv.Load()

	port := strings.TrimSpace(os.Getenv("PORT"))
	if port == "" {
		port = "5000"
	}

	mongoURI := strings.TrimSpace(os.Getenv("MONGO_URI"))
	if mongoURI == "" {
		return Config{}, fmt.Errorf("environment variable MONGO_URI is required")
	}

	mongoDBName := strings.TrimSpace(os.Getenv("MONGO_DB_NAME"))
	if mongoDBName == "" {
		return Config{}, fmt.Errorf("environment variable MONGO_DB_NAME is required")
	}

	jwtSecret := strings.TrimSpace(os.Getenv("JWT_SECRET"))
	if jwtSecret == "" {
		return Config{}, fmt.Errorf("environment variable JWT_SECRET is required")
	}

	return Config{
		Port:          port,
		MongoURI:      mongoURI,
		MongoDatabase: mongoDBName,
		JWTSecret:     jwtSecret,
	}, nil
}
