package config

import (
	"fmt"
	"os"
	"strings"

	"github.com/joho/godotenv"
)

type Config struct {
	Port              string
	MongoURI          string
	MongoDatabase     string
	JWTSecret         string
	R2AccountID       string
	R2AccessKeyID     string
	R2SecretAccessKey string
	R2BucketName      string
	R2PublicURL       string
}

func (c Config) HasR2() bool {
	return c.R2AccountID != "" && c.R2AccessKeyID != "" && c.R2SecretAccessKey != "" && c.R2BucketName != ""
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

	r2AccountID := strings.TrimSpace(os.Getenv("R2_ACCOUNT_ID"))
	r2AccessKeyID := strings.TrimSpace(os.Getenv("R2_ACCESS_KEY_ID"))
	r2SecretAccessKey := strings.TrimSpace(os.Getenv("R2_SECRET_ACCESS_KEY"))
	r2BucketName := strings.TrimSpace(os.Getenv("R2_BUCKET_NAME"))
	r2PublicURL := strings.TrimSpace(os.Getenv("R2_PUBLIC_URL"))

	return Config{
		Port:              port,
		MongoURI:          mongoURI,
		MongoDatabase:     mongoDBName,
		JWTSecret:         jwtSecret,
		R2AccountID:       r2AccountID,
		R2AccessKeyID:     r2AccessKeyID,
		R2SecretAccessKey: r2SecretAccessKey,
		R2BucketName:      r2BucketName,
		R2PublicURL:       r2PublicURL,
	}, nil
}
