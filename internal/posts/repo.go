package posts

import (
	"context"
	"errors"
	"fmt"
	"strings"
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
		coll: db.Collection("posts"),
	}
}

func (r *Repo) CreatePost(ctx context.Context, post Post) (Post, error) {
	res, err := r.coll.InsertOne(ctx, post)

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

func (r *Repo) ListPublished(ctx context.Context, nextCursor string, limit int64, filter PostFilter) ([]Post, error) {

	query := bson.M{}

	query["status"] = StatusPublished

	if nextCursor != "" {
		objID, err := primitive.ObjectIDFromHex(nextCursor)

		if err != nil {
			return []Post{}, fmt.Errorf("Invalid parameter for next cursor: %w", err)
		}

		query["_id"] = bson.M{"$lt": objID}
	}

	if strings.TrimSpace(filter.Search) != "" {
		searchStr := strings.TrimSpace(filter.Search)

		query["title"] = bson.M{"$regex": searchStr, "$options": "i"}
	}

	if strings.TrimSpace(filter.Tag) != "" {
		query["tags"] = strings.TrimSpace(filter.Tag)
	}

	opts := options.Find()

	opts.SetSort(bson.D{{Key: "_id", Value: -1}})

	if limit <= 0 {
		limit = 10
	}

	opts.SetLimit(limit)

	inCtx, cancel := context.WithTimeout(ctx, time.Second*5)
	defer cancel()

	cursor, err := r.coll.Find(inCtx, query, opts)

	if err != nil {
		return []Post{}, fmt.Errorf("Failed to find posts: %w", err)
	}
	defer cursor.Close(inCtx)

	var posts []Post
	if err := cursor.All(inCtx, &posts); err != nil {
		return []Post{}, fmt.Errorf("Failed to decode posts: %w", err)
	}

	if posts == nil {
		posts = []Post{}
	}

	return posts, nil
}

func (r *Repo) ListByAuthor(ctx context.Context, authorID primitive.ObjectID, nextCursor string, limit int64) ([]Post, error) {
	query := bson.M{}

	query["author_id"] = authorID

	if nextCursor != "" {
		objID, err := primitive.ObjectIDFromHex(nextCursor)

		if err != nil {
			return []Post{}, fmt.Errorf("Invalid value for next cursor: %w", err)
		}

		query["_id"] = bson.M{"$lt": objID}
	}

	opts := options.Find()

	if limit <= 0 {
		limit = 10
	}
	opts.SetLimit(limit)
	opts.SetSort(bson.D{{Key: "_id", Value: -1}})

	inCtx, cancel := context.WithTimeout(ctx, time.Second*5)
	defer cancel()

	cursor, err := r.coll.Find(inCtx, query, opts)
	if err != nil {
		return []Post{}, fmt.Errorf("Failed to find posts: %w", err)
	}

	defer cursor.Close(inCtx)

	var posts []Post

	if err := cursor.All(inCtx, &posts); err != nil {
		return []Post{}, fmt.Errorf("Failed to decode posts: %w", err)
	}

	if posts == nil {
		posts = []Post{}
	}

	return posts, nil
}

func (r *Repo) ListAllAdmin(ctx context.Context, nextCursor string, maxLimit int64, filter PostFilter) ([]Post, error) {
	query := bson.M{}

	if strings.TrimSpace(filter.Status) != "" {
		query["status"] = strings.TrimSpace(filter.Status)
	}

	if strings.TrimSpace(filter.Search) != "" {
		query["title"] = bson.M{"$regex": strings.TrimSpace(filter.Search), "$options": "i"}
	}

	if strings.TrimSpace(filter.Tag) != "" {
		query["tags"] = strings.TrimSpace(filter.Tag)
	}

	if nextCursor != "" {
		objId, err := primitive.ObjectIDFromHex(nextCursor)

		if err != nil {
			return []Post{}, fmt.Errorf("Invalid value for next cursor: %w", err)
		}

		query["_id"] = bson.M{"$lt": objId}
	}

	opts := options.Find()

	if maxLimit <= 0 {
		maxLimit = 10
	}

	opts.SetLimit(maxLimit)
	opts.SetSort(bson.D{{Key: "_id", Value: -1}})

	inCtx, cancel := context.WithTimeout(ctx, time.Second*5)
	defer cancel()

	cursor, err := r.coll.Find(inCtx, query, opts)
	if err != nil {
		return []Post{}, fmt.Errorf("Failed to find posts: %w", err)
	}
	defer cursor.Close(inCtx)

	var posts []Post

	if err := cursor.All(inCtx, &posts); err != nil {
		return []Post{}, fmt.Errorf("Failed to decode posts: %w", err)
	}

	if posts == nil {
		return []Post{}, nil
	}

	return posts, nil
}

func (r *Repo) GetByID(ctx context.Context, postID primitive.ObjectID) (Post, error) {
	query := bson.M{"_id": postID}

	inCtx, cancel := context.WithTimeout(ctx, time.Second*5)
	defer cancel()

	var post Post

	err := r.coll.FindOne(inCtx, query).Decode(&post)

	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return Post{}, mongo.ErrNoDocuments
		}

		return Post{}, fmt.Errorf("failed to find post: %w", err)
	}

	return post, nil
}

func (r *Repo) Delete(ctx context.Context, postID primitive.ObjectID, authorID *primitive.ObjectID) error {
	query := bson.M{"_id": postID}

	if authorID != nil {
		query["author_id"] = *authorID
	}

	inCtx, cancel := context.WithTimeout(ctx, time.Second*5)
	defer cancel()

	result, err := r.coll.DeleteOne(inCtx, query)

	if err != nil {
		return fmt.Errorf("Failed to delete post: %w", err)
	}

	if result.DeletedCount == 0 {
		return mongo.ErrNoDocuments
	}

	return nil
}
