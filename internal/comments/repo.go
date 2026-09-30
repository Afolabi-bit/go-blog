package comments

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

func (r *Repo) ListRootComments(ctx context.Context, postID primitive.ObjectID, nextCursor string, limit int64) ([]Comment, error) {
	inCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	query := bson.M{
		"post_id": postID,
		"$or": []bson.M{
			{"parent_id": nil},
			{"parent_id": bson.M{"$exists": false}},
		},
	}

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
		return nil, fmt.Errorf("failed to list root comments: %w", err)
	}
	defer cursor.Close(inCtx)

	var comments []Comment
	if err := cursor.All(inCtx, &comments); err != nil {
		return nil, fmt.Errorf("failed to decode root comments: %w", err)
	}

	if comments == nil {
		comments = []Comment{}
	}

	return comments, nil
}

func (r *Repo) ListReplies(ctx context.Context, postID primitive.ObjectID, parentIDs []primitive.ObjectID) ([]Comment, error) {
	if len(parentIDs) == 0 {
		return []Comment{}, nil
	}

	inCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	query := bson.M{
		"post_id":   postID,
		"parent_id": bson.M{"$in": parentIDs},
	}

	opts := options.Find().SetSort(bson.D{{Key: "created_at", Value: 1}})

	cursor, err := r.coll.Find(inCtx, query, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to list replies: %w", err)
	}
	defer cursor.Close(inCtx)

	var replies []Comment
	if err := cursor.All(inCtx, &replies); err != nil {
		return nil, fmt.Errorf("failed to decode replies: %w", err)
	}

	if replies == nil {
		replies = []Comment{}
	}

	return replies, nil
}

func (r *Repo) ListAllAdmin(ctx context.Context, nextCursor string, limit int64, filter AdminCommentFilter) ([]Comment, error) {
	inCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	query := bson.M{}

	if strings.TrimSpace(filter.PostID) != "" {
		if pid, err := primitive.ObjectIDFromHex(strings.TrimSpace(filter.PostID)); err == nil {
			query["post_id"] = pid
		}
	}

	if strings.TrimSpace(filter.AuthorID) != "" {
		if aid, err := primitive.ObjectIDFromHex(strings.TrimSpace(filter.AuthorID)); err == nil {
			query["author_id"] = aid
		}
	}

	if strings.TrimSpace(filter.Search) != "" {
		query["content"] = bson.M{"$regex": strings.TrimSpace(filter.Search), "$options": "i"}
	}

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
		return nil, fmt.Errorf("failed to list comments for admin: %w", err)
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

func (r *Repo) DeleteRepliesByParentID(ctx context.Context, parentID primitive.ObjectID) (int64, error) {
	inCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	res, err := r.coll.DeleteMany(inCtx, bson.M{"parent_id": parentID})
	if err != nil {
		return 0, fmt.Errorf("failed to delete child replies: %w", err)
	}

	return res.DeletedCount, nil
}

