package db

import (
	"context"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// EnsureIndexes creates required indexes across all collections if they don't already exist.
func EnsureIndexes(ctx context.Context, database *mongo.Database) error {
	inCtx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()

	// 1. Users collection indexes
	usersColl := database.Collection("users")
	userIndexes := []mongo.IndexModel{
		{
			Keys:    bson.D{{Key: "email", Value: 1}},
			Options: options.Index().SetUnique(true).SetName("idx_users_email_unique"),
		},
	}
	if _, err := usersColl.Indexes().CreateMany(inCtx, userIndexes); err != nil {
		return fmt.Errorf("failed to create users indexes: %w", err)
	}

	// 2. Posts collection indexes
	postsColl := database.Collection("posts")
	postIndexes := []mongo.IndexModel{
		{
			Keys:    bson.D{{Key: "slug", Value: 1}},
			Options: options.Index().SetUnique(true).SetSparse(true).SetName("idx_posts_slug_unique"),
		},
		{
			Keys:    bson.D{{Key: "status", Value: 1}, {Key: "_id", Value: -1}},
			Options: options.Index().SetName("idx_posts_status_id"),
		},
		{
			Keys:    bson.D{{Key: "author_id", Value: 1}, {Key: "_id", Value: -1}},
			Options: options.Index().SetName("idx_posts_author_id"),
		},
		{
			Keys: bson.D{
				{Key: "title", Value: "text"},
				{Key: "content", Value: "text"},
			},
			Options: options.Index().SetName("idx_posts_text_search"),
		},
	}
	if _, err := postsColl.Indexes().CreateMany(inCtx, postIndexes); err != nil {
		return fmt.Errorf("failed to create posts indexes: %w", err)
	}

	// 3. Author Requests collection indexes
	authorReqColl := database.Collection("author_requests")
	authorReqIndexes := []mongo.IndexModel{
		{
			Keys:    bson.D{{Key: "user_id", Value: 1}, {Key: "status", Value: 1}},
			Options: options.Index().SetName("idx_author_requests_user_status"),
		},
		{
			Keys:    bson.D{{Key: "status", Value: 1}, {Key: "_id", Value: -1}},
			Options: options.Index().SetName("idx_author_requests_status_id"),
		},
	}
	if _, err := authorReqColl.Indexes().CreateMany(inCtx, authorReqIndexes); err != nil {
		return fmt.Errorf("failed to create author_requests indexes: %w", err)
	}

	return nil
}
