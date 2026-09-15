package posts

import (
	"context"
	"fmt"

	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

type Repo struct {
	Collection *mongo.Collection
}

func NewRepo(db *mongo.Database) *Repo {
	return &Repo{
		Collection: db.Collection("posts"),
	}
}

func (r *Repo) CreatePost(ctx context.Context, post Post) (Post, error) {
	res, err := r.Collection.InsertOne(ctx, post)

	if err != nil {
		return Post{}, fmt.Errorf("failed to insert post: %w", err)
	}

	id, ok := res.InsertedID.(primitive.ObjectID)
	if !ok {
		return Post{}, fmt.Errorf("post ID is not an ObjectID")
	}

	post.ID = id

	return post, nil
}
