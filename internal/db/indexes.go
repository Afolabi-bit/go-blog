package db

import (
	"context"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
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
	_ = deduplicatePostSlugs(inCtx, postsColl)

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

	// 4. Comments collection indexes
	commentsColl := database.Collection("comments")
	commentIndexes := []mongo.IndexModel{
		{
			Keys:    bson.D{{Key: "post_id", Value: 1}, {Key: "created_at", Value: -1}},
			Options: options.Index().SetName("idx_comments_post_created"),
		},
	}
	if _, err := commentsColl.Indexes().CreateMany(inCtx, commentIndexes); err != nil {
		return fmt.Errorf("failed to create comments indexes: %w", err)
	}

	// 5. Likes collection indexes
	likesColl := database.Collection("likes")
	likeIndexes := []mongo.IndexModel{
		{
			Keys:    bson.D{{Key: "post_id", Value: 1}, {Key: "user_id", Value: 1}},
			Options: options.Index().SetUnique(true).SetName("idx_likes_post_user_unique"),
		},
		{
			Keys:    bson.D{{Key: "post_id", Value: 1}},
			Options: options.Index().SetName("idx_likes_post_id"),
		},
	}
	if _, err := likesColl.Indexes().CreateMany(inCtx, likeIndexes); err != nil {
		return fmt.Errorf("failed to create likes indexes: %w", err)
	}

	// 6. Refresh tokens collection indexes
	refreshTokensColl := database.Collection("refresh_tokens")
	ttlSeconds := int32(0)
	refreshTokenIndexes := []mongo.IndexModel{
		{
			Keys:    bson.D{{Key: "token_hash", Value: 1}},
			Options: options.Index().SetUnique(true).SetName("idx_refresh_tokens_hash_unique"),
		},
		{
			Keys:    bson.D{{Key: "user_id", Value: 1}},
			Options: options.Index().SetName("idx_refresh_tokens_user_id"),
		},
		{
			Keys:    bson.D{{Key: "expires_at", Value: 1}},
			Options: options.Index().SetExpireAfterSeconds(ttlSeconds).SetName("idx_refresh_tokens_ttl"),
		},
	}
	if _, err := refreshTokensColl.Indexes().CreateMany(inCtx, refreshTokenIndexes); err != nil {
		return fmt.Errorf("failed to create refresh_tokens indexes: %w", err)
	}

	return nil
}

// deduplicatePostSlugs ensures all existing posts have distinct slugs before building the unique index.
func deduplicatePostSlugs(ctx context.Context, postsColl *mongo.Collection) error {
	pipeline := mongo.Pipeline{
		{{Key: "$group", Value: bson.D{
			{Key: "_id", Value: "$slug"},
			{Key: "count", Value: bson.D{{Key: "$sum", Value: 1}}},
			{Key: "ids", Value: bson.D{{Key: "$push", Value: "$_id"}}},
		}}},
		{{Key: "$match", Value: bson.D{
			{Key: "_id", Value: bson.D{{Key: "$nin", Value: bson.A{nil, ""}}}},
			{Key: "count", Value: bson.D{{Key: "$gt", Value: 1}}},
		}}},
	}

	cursor, err := postsColl.Aggregate(ctx, pipeline)
	if err != nil {
		return err
	}
	defer cursor.Close(ctx)

	for cursor.Next(ctx) {
		var group struct {
			Slug string               `bson:"_id"`
			IDs  []primitive.ObjectID `bson:"ids"`
		}
		if err := cursor.Decode(&group); err != nil {
			return err
		}

		for i := 1; i < len(group.IDs); i++ {
			newSlug := fmt.Sprintf("%s-%d", group.Slug, i)
			_, _ = postsColl.UpdateOne(ctx,
				bson.M{"_id": group.IDs[i]},
				bson.M{"$set": bson.M{"slug": newSlug}},
			)
		}
	}
	return cursor.Err()
}
