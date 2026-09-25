package auth

import (
	"context"
	"errors"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

type SessionRepo struct {
	coll *mongo.Collection
}

func NewSessionRepo(db *mongo.Database) *SessionRepo {
	return &SessionRepo{
		coll: db.Collection("refresh_tokens"),
	}
}

func (r *SessionRepo) StoreToken(ctx context.Context, userID primitive.ObjectID, rawToken string, ttl time.Duration) error {
	inCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	now := time.Now()
	rt := RefreshToken{
		UserID:    userID,
		TokenHash: HashToken(rawToken),
		ExpiresAt: now.Add(ttl),
		Revoked:   false,
		CreatedAt: now,
	}

	_, err := r.coll.InsertOne(inCtx, rt)
	if err != nil {
		return fmt.Errorf("failed to store refresh token: %w", err)
	}
	return nil
}

func (r *SessionRepo) ValidateAndRotate(ctx context.Context, rawToken string, newRawToken string, newTTL time.Duration) (primitive.ObjectID, error) {
	inCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	tokenHash := HashToken(rawToken)
	filter := bson.M{
		"token_hash": tokenHash,
		"revoked":    false,
		"expires_at": bson.M{"$gt": time.Now()},
	}

	update := bson.M{
		"$set": bson.M{
			"revoked": true,
		},
	}

	var oldToken RefreshToken
	err := r.coll.FindOneAndUpdate(inCtx, filter, update).Decode(&oldToken)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return primitive.NilObjectID, ErrInvalidToken
		}
		return primitive.NilObjectID, fmt.Errorf("failed to validate refresh token: %w", err)
	}

	if err := r.StoreToken(ctx, oldToken.UserID, newRawToken, newTTL); err != nil {
		return primitive.NilObjectID, err
	}

	return oldToken.UserID, nil
}

func (r *SessionRepo) RevokeToken(ctx context.Context, rawToken string) error {
	inCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	filter := bson.M{"token_hash": HashToken(rawToken)}
	update := bson.M{"$set": bson.M{"revoked": true}}

	_, err := r.coll.UpdateOne(inCtx, filter, update)
	return err
}

func (r *SessionRepo) RevokeAllForUser(ctx context.Context, userID primitive.ObjectID) error {
	inCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	filter := bson.M{"user_id": userID}
	update := bson.M{"$set": bson.M{"revoked": true}}

	_, err := r.coll.UpdateMany(inCtx, filter, update)
	return err
}
