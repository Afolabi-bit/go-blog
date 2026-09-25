package likes

import (
	"blog-api/internal/posts"
	"context"
	"errors"
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

type Repository interface {
	FindLike(ctx context.Context, postID, userID primitive.ObjectID) (bool, error)
	CreateLike(ctx context.Context, like Like) error
	DeleteLike(ctx context.Context, postID, userID primitive.ObjectID) error
}

type PostRepo interface {
	GetByID(ctx context.Context, postID primitive.ObjectID) (posts.Post, error)
	IncrementLikesCount(ctx context.Context, postID primitive.ObjectID, delta int64) error
}

type Service struct {
	repo     Repository
	postRepo PostRepo
}

func NewService(repo Repository, postRepo PostRepo) *Service {
	return &Service{
		repo:     repo,
		postRepo: postRepo,
	}
}

func (s *Service) ToggleLike(ctx context.Context, postIDStr, userIDStr string) (LikeStatus, error) {
	postID, err := primitive.ObjectIDFromHex(postIDStr)
	if err != nil {
		return LikeStatus{}, ErrInvalidID
	}

	userID, err := primitive.ObjectIDFromHex(userIDStr)
	if err != nil {
		return LikeStatus{}, ErrInvalidID
	}

	post, err := s.postRepo.GetByID(ctx, postID)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return LikeStatus{}, ErrPostNotFound
		}
		return LikeStatus{}, err
	}

	if post.Status != posts.StatusPublished {
		return LikeStatus{}, ErrPostNotFound
	}

	liked, err := s.repo.FindLike(ctx, postID, userID)
	if err != nil {
		return LikeStatus{}, err
	}

	if liked {
		// Unlike
		if err := s.repo.DeleteLike(ctx, postID, userID); err != nil {
			return LikeStatus{}, err
		}
		_ = s.postRepo.IncrementLikesCount(ctx, postID, -1)

		newCount := post.LikesCount - 1
		if newCount < 0 {
			newCount = 0
		}
		return LikeStatus{Liked: false, LikesCount: newCount}, nil
	}

	// Like
	newLike := Like{
		PostID:    postID,
		UserID:    userID,
		CreatedAt: time.Now(),
	}
	if err := s.repo.CreateLike(ctx, newLike); err != nil {
		return LikeStatus{}, err
	}
	_ = s.postRepo.IncrementLikesCount(ctx, postID, 1)

	return LikeStatus{Liked: true, LikesCount: post.LikesCount + 1}, nil
}

func (s *Service) GetStatus(ctx context.Context, postIDStr string, userIDStr *string) (LikeStatus, error) {
	postID, err := primitive.ObjectIDFromHex(postIDStr)
	if err != nil {
		return LikeStatus{}, ErrInvalidID
	}

	post, err := s.postRepo.GetByID(ctx, postID)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return LikeStatus{}, ErrPostNotFound
		}
		return LikeStatus{}, err
	}

	liked := false
	if userIDStr != nil {
		if uID, err := primitive.ObjectIDFromHex(*userIDStr); err == nil {
			liked, _ = s.repo.FindLike(ctx, postID, uID)
		}
	}

	return LikeStatus{
		Liked:      liked,
		LikesCount: post.LikesCount,
	}, nil
}
