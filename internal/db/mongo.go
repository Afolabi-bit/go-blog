package db

import (
	"blog-api/internal/config"
	"context"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type Mongo struct {
	Client   *mongo.Client
	Database *mongo.Database
}

func Connect(ctx context.Context, cfg config.Config) (*Mongo, error) {
	clientOptions := options.Client().ApplyURI(cfg.MongoURI)

	client, err := mongo.Connect(ctx, clientOptions)

	if err != nil {
		return nil, fmt.Errorf("Failed to connect to database: %v", err)
	}

	pingCtx, cancel := context.WithTimeout(ctx, time.Second*10)
	defer cancel()

	if err = client.Ping(pingCtx, nil); err != nil {
		return nil, fmt.Errorf("MongoDB ping failed: %v", err)
	}

	database := client.Database(cfg.MongoDatabase)

	return &Mongo{
		Client:   client,
		Database: database,
	}, nil
}

func (m *Mongo) Disconnect(ctx context.Context) error {
	if m.Client == nil {
		return nil
	}

	disconnectCtx, cancel := context.WithTimeout(ctx, time.Second*5)
	defer cancel()

	if err := m.Client.Disconnect(disconnectCtx); err != nil {
		return fmt.Errorf("MongoDB disconnection failed: %v", err)
	}
	return nil
}
