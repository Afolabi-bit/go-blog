package user

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
	collection *mongo.Collection
}

func NewRepo(db *mongo.Database) *Repo {
	return &Repo{
		collection: db.Collection("users"),
	}
}

func (r *Repo) FindUserByEmail(ctx context.Context, email string) (User, error) {
	var user User

	email = strings.ToLower(strings.TrimSpace(email))

	filter := bson.M{"email": email}

	err := r.collection.FindOne(ctx, filter).Decode(&user)

	if err != nil {
		if errors.Is(mongo.ErrNoDocuments, err) {
			return User{}, mongo.ErrNoDocuments
		}
		return User{}, fmt.Errorf("find user by email failed: %w", err)
	}

	return user, nil
}

func (r *Repo) FinduserByID(ctx context.Context, id string) (User, error) {
	var user User

	idObj, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return User{}, fmt.Errorf("Find user by ID failed: %v", err)
	}

	filter := bson.M{"_id": idObj}

	err = r.collection.FindOne(ctx, filter).Decode(&user)

	if err != nil {
		if errors.Is(mongo.ErrNoDocuments, err) {
			return User{}, mongo.ErrNoDocuments
		}
		return User{}, fmt.Errorf("find user by ID failed: %w", err)
	}

	return user, nil
}

func (r *Repo) CreateUser(ctx context.Context, user User) (User, error) {
	res, err := r.collection.InsertOne(ctx, user)

	if err != nil {
		return User{}, fmt.Errorf("Insert user failed: %w", err)
	}

	id, ok := res.InsertedID.(primitive.ObjectID)

	if !ok {
		return User{}, fmt.Errorf("Insert user failed: invalid ID type from insert result")
	}

	user.ID = id
	return user, nil
}

func (r *Repo) UpdateRole(ctx context.Context, userID primitive.ObjectID, newRole string) error {
	filter := bson.M{"_id": userID}
	update := bson.M{
		"$set": bson.M{
			"role":       newRole,
			"updated_at": time.Now(),
		},
	}

	result, err := r.collection.UpdateOne(ctx, filter, update)
	if err != nil {
		return fmt.Errorf("update user role failed: %w", err)
	}

	if result.MatchedCount == 0 {
		return mongo.ErrNoDocuments
	}

	return nil
}

func (r *Repo) UpdateProfile(ctx context.Context, userID primitive.ObjectID, req UpdateProfileRequest) (User, error) {
	inCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	fields := bson.M{
		"updated_at": time.Now(),
	}

	if req.FirstName != nil && strings.TrimSpace(*req.FirstName) != "" {
		fields["first_name"] = strings.TrimSpace(*req.FirstName)
	}
	if req.LastName != nil && strings.TrimSpace(*req.LastName) != "" {
		fields["last_name"] = strings.TrimSpace(*req.LastName)
	}
	if req.Bio != nil {
		fields["bio"] = strings.TrimSpace(*req.Bio)
	}
	if req.AvatarURL != nil {
		fields["avatar_url"] = strings.TrimSpace(*req.AvatarURL)
	}

	filter := bson.M{"_id": userID}
	update := bson.M{"$set": fields}

	opts := options.FindOneAndUpdate().SetReturnDocument(options.After)
	var updated User
	err := r.collection.FindOneAndUpdate(inCtx, filter, update, opts).Decode(&updated)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return User{}, mongo.ErrNoDocuments
		}
		return User{}, fmt.Errorf("update profile failed: %w", err)
	}

	return updated, nil
}

func (r *Repo) UpdatePassword(ctx context.Context, userID primitive.ObjectID, newPasswordHash string) error {
	inCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	filter := bson.M{"_id": userID}
	update := bson.M{
		"$set": bson.M{
			"password_hash": newPasswordHash,
			"updated_at":    time.Now(),
		},
	}

	res, err := r.collection.UpdateOne(inCtx, filter, update)
	if err != nil {
		return fmt.Errorf("update password failed: %w", err)
	}

	if res.MatchedCount == 0 {
		return mongo.ErrNoDocuments
	}

	return nil
}
