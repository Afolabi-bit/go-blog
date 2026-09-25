package authorrequest

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
		coll: db.Collection("author_requests"),
	}
}

func (r *Repo) Create(ctx context.Context, req AuthorRequest) (AuthorRequest, error) {
	inCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	res, err := r.coll.InsertOne(inCtx, req)
	if err != nil {
		return AuthorRequest{}, fmt.Errorf("failed to insert author request: %w", err)
	}

	id, ok := res.InsertedID.(primitive.ObjectID)
	if !ok {
		return AuthorRequest{}, fmt.Errorf("author request ID is not an ObjectID")
	}

	req.ID = id
	return req, nil
}

func (r *Repo) FindPendingByUserID(ctx context.Context, userID primitive.ObjectID) (AuthorRequest, error) {
	inCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	filter := bson.M{
		"user_id": userID,
		"status":  StatusPending,
	}

	var req AuthorRequest
	err := r.coll.FindOne(inCtx, filter).Decode(&req)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return AuthorRequest{}, mongo.ErrNoDocuments
		}
		return AuthorRequest{}, fmt.Errorf("failed to find pending author request: %w", err)
	}

	return req, nil
}

func (r *Repo) FindLatestByUserID(ctx context.Context, userID primitive.ObjectID) (AuthorRequest, error) {
	inCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	filter := bson.M{"user_id": userID}
	opts := options.FindOne().SetSort(bson.D{{Key: "_id", Value: -1}})

	var req AuthorRequest
	err := r.coll.FindOne(inCtx, filter, opts).Decode(&req)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return AuthorRequest{}, mongo.ErrNoDocuments
		}
		return AuthorRequest{}, fmt.Errorf("failed to find latest author request: %w", err)
	}

	return req, nil
}

func (r *Repo) FindByID(ctx context.Context, id primitive.ObjectID) (AuthorRequest, error) {
	inCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	filter := bson.M{"_id": id}

	var req AuthorRequest
	err := r.coll.FindOne(inCtx, filter).Decode(&req)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return AuthorRequest{}, mongo.ErrNoDocuments
		}
		return AuthorRequest{}, fmt.Errorf("failed to find author request by ID: %w", err)
	}

	return req, nil
}

func (r *Repo) List(ctx context.Context, nextCursor string, limit int64, status string) ([]AuthorRequest, error) {
	inCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	query := bson.M{}

	if strings.TrimSpace(status) != "" {
		query["status"] = strings.TrimSpace(status)
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
		return nil, fmt.Errorf("failed to list author requests: %w", err)
	}
	defer cursor.Close(inCtx)

	var requests []AuthorRequest
	if err := cursor.All(inCtx, &requests); err != nil {
		return nil, fmt.Errorf("failed to decode author requests: %w", err)
	}

	if requests == nil {
		requests = []AuthorRequest{}
	}

	return requests, nil
}

func (r *Repo) UpdateReview(ctx context.Context, id primitive.ObjectID, reviewerID primitive.ObjectID, status string, notes string) (AuthorRequest, error) {
	inCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	filter := bson.M{
		"_id": id,
	}

	now := time.Now()
	update := bson.M{
		"$set": bson.M{
			"status":       status,
			"reviewed_by":  reviewerID,
			"review_notes": strings.TrimSpace(notes),
			"updated_at":   now,
		},
	}

	opts := options.FindOneAndUpdate().SetReturnDocument(options.After)

	var updated AuthorRequest
	err := r.coll.FindOneAndUpdate(inCtx, filter, update, opts).Decode(&updated)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return AuthorRequest{}, mongo.ErrNoDocuments
		}
		return AuthorRequest{}, fmt.Errorf("failed to update author request review: %w", err)
	}

	return updated, nil
}
