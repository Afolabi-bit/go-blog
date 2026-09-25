package likes_test

import (
	"blog-api/internal/likes"
	"blog-api/internal/posts"
	"context"
	"errors"
	"testing"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type mockLikeRepo struct {
	findLikeFunc   func(ctx context.Context, postID, userID primitive.ObjectID) (bool, error)
	createLikeFunc func(ctx context.Context, like likes.Like) error
	deleteLikeFunc func(ctx context.Context, postID, userID primitive.ObjectID) error
}

func (m *mockLikeRepo) FindLike(ctx context.Context, postID, userID primitive.ObjectID) (bool, error) {
	if m.findLikeFunc != nil {
		return m.findLikeFunc(ctx, postID, userID)
	}
	return false, nil
}

func (m *mockLikeRepo) CreateLike(ctx context.Context, like likes.Like) error {
	if m.createLikeFunc != nil {
		return m.createLikeFunc(ctx, like)
	}
	return nil
}

func (m *mockLikeRepo) DeleteLike(ctx context.Context, postID, userID primitive.ObjectID) error {
	if m.deleteLikeFunc != nil {
		return m.deleteLikeFunc(ctx, postID, userID)
	}
	return nil
}

type mockPostRepo struct {
	getByIDFunc        func(ctx context.Context, postID primitive.ObjectID) (posts.Post, error)
	incrementLikesFunc func(ctx context.Context, postID primitive.ObjectID, delta int64) error
}

func (m *mockPostRepo) GetByID(ctx context.Context, postID primitive.ObjectID) (posts.Post, error) {
	if m.getByIDFunc != nil {
		return m.getByIDFunc(ctx, postID)
	}
	return posts.Post{
		ID:         postID,
		Status:     posts.StatusPublished,
		LikesCount: 5,
	}, nil
}

func (m *mockPostRepo) IncrementLikesCount(ctx context.Context, postID primitive.ObjectID, delta int64) error {
	if m.incrementLikesFunc != nil {
		return m.incrementLikesFunc(ctx, postID, delta)
	}
	return nil
}

func TestToggleLike_Like(t *testing.T) {
	postID := primitive.NewObjectID()
	userID := primitive.NewObjectID()

	likeCreated := false
	likeRepo := &mockLikeRepo{
		findLikeFunc: func(ctx context.Context, p, u primitive.ObjectID) (bool, error) {
			return false, nil
		},
		createLikeFunc: func(ctx context.Context, l likes.Like) error {
			likeCreated = true
			return nil
		},
	}

	postRepo := &mockPostRepo{}
	svc := likes.NewService(likeRepo, postRepo)

	res, err := svc.ToggleLike(context.Background(), postID.Hex(), userID.Hex())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !res.Liked {
		t.Errorf("expected liked to be true")
	}
	if res.LikesCount != 6 {
		t.Errorf("expected likes count 6, got %d", res.LikesCount)
	}
	if !likeCreated {
		t.Errorf("expected like to be created in repo")
	}
}

func TestToggleLike_Unlike(t *testing.T) {
	postID := primitive.NewObjectID()
	userID := primitive.NewObjectID()

	likeDeleted := false
	likeRepo := &mockLikeRepo{
		findLikeFunc: func(ctx context.Context, p, u primitive.ObjectID) (bool, error) {
			return true, nil
		},
		deleteLikeFunc: func(ctx context.Context, p, u primitive.ObjectID) error {
			likeDeleted = true
			return nil
		},
	}

	postRepo := &mockPostRepo{}
	svc := likes.NewService(likeRepo, postRepo)

	res, err := svc.ToggleLike(context.Background(), postID.Hex(), userID.Hex())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if res.Liked {
		t.Errorf("expected liked to be false")
	}
	if res.LikesCount != 4 {
		t.Errorf("expected likes count 4, got %d", res.LikesCount)
	}
	if !likeDeleted {
		t.Errorf("expected like to be deleted from repo")
	}
}

func TestToggleLike_UnpublishedPost(t *testing.T) {
	postID := primitive.NewObjectID()
	userID := primitive.NewObjectID()

	postRepo := &mockPostRepo{
		getByIDFunc: func(ctx context.Context, p primitive.ObjectID) (posts.Post, error) {
			return posts.Post{
				ID:     p,
				Status: posts.StatusDraft,
			}, nil
		},
	}

	svc := likes.NewService(&mockLikeRepo{}, postRepo)

	_, err := svc.ToggleLike(context.Background(), postID.Hex(), userID.Hex())
	if !errors.Is(err, likes.ErrPostNotFound) {
		t.Fatalf("expected ErrPostNotFound, got: %v", err)
	}
}
