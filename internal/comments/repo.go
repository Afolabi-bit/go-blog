package comments

import (
	"context"
	"errors"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type Repo struct {
	coll *mongo.Collection
}

func NewRepo(db *mongo.Database) *Repo {
	return &Repo{
		coll: db.Collection("comments"),
	}
}

func (r *Repo) Create(ctx context.Context, comment Comment) (Comment, error) {
	inCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	res, err := r.coll.InsertOne(inCtx, comment)
	if err != nil {
		return Comment{}, fmt.Errorf("failed to insert comment: %w", err)
	}

	id, ok := res.InsertedID.(primitive.ObjectID)
	if !ok {
		return Comment{}, fmt.Errorf("comment ID is not an ObjectID")
	}

	comment.ID = id
	return comment, nil
}

func (r *Repo) FindByID(ctx context.Context, id primitive.ObjectID) (Comment, error) {
	inCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	filter := bson.M{"_id": id}

	var comment Comment
	err := r.coll.FindOne(inCtx, filter).Decode(&comment)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return Comment{}, mongo.ErrNoDocuments
		}
		return Comment{}, fmt.Errorf("failed to find comment: %w", err)
	}

	return comment, nil
}

func (r *Repo) ListByPostID(ctx context.Context, postID primitive.ObjectID, nextCursor string, limit int64) ([]Comment, error) {
	inCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	query := bson.M{"post_id": postID}

	if nextCursor != "" {
		objID, err := primitive.ObjectIDFromHex(nextCursor)
		if err != nil {
			return nil, fmt.Errorf("invalid cursor: %w", err)
		}
		query["_id"] = bson.M{"$lt": objID}
	}

	opts := options.Find().
		SetSort(bson.D{{Key: "_id", Value: -1}}).
		SetLimit(limit)

	cursor, err := r.coll.Find(inCtx, query, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to list comments: %w", err)
	}
	defer cursor.Close(inCtx)

	var comments []Comment
	if err := cursor.All(inCtx, &comments); err != nil {
		return nil, fmt.Errorf("failed to decode comments: %w", err)
	}

	if comments == nil {
		comments = []Comment{}
	}

	return comments, nil
}

func (r *Repo) Delete(ctx context.Context, id primitive.ObjectID) error {
	inCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	res, err := r.coll.DeleteOne(inCtx, bson.M{"_id": id})
	if err != nil {
		return fmt.Errorf("failed to delete comment: %w", err)
	}

	if res.DeletedCount == 0 {
		return mongo.ErrNoDocuments
	}

	return nil
}
