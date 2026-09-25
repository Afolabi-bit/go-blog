package likes

import (
	"context"
	"errors"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

type Repo struct {
	coll *mongo.Collection
}

func NewRepo(db *mongo.Database) *Repo {
	return &Repo{
		coll: db.Collection("likes"),
	}
}

func (r *Repo) FindLike(ctx context.Context, postID, userID primitive.ObjectID) (bool, error) {
	inCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	filter := bson.M{
		"post_id": postID,
		"user_id": userID,
	}

	var like Like
	err := r.coll.FindOne(inCtx, filter).Decode(&like)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return false, nil
		}
		return false, fmt.Errorf("failed to find like: %w", err)
	}

	return true, nil
}

func (r *Repo) CreateLike(ctx context.Context, like Like) error {
	inCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	_, err := r.coll.InsertOne(inCtx, like)
	if err != nil {
		return fmt.Errorf("failed to insert like: %w", err)
	}

	return nil
}

func (r *Repo) DeleteLike(ctx context.Context, postID, userID primitive.ObjectID) error {
	inCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	filter := bson.M{
		"post_id": postID,
		"user_id": userID,
	}

	_, err := r.coll.DeleteOne(inCtx, filter)
	if err != nil {
		return fmt.Errorf("failed to delete like: %w", err)
	}

	return nil
}
