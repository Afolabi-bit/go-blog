package user

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
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
