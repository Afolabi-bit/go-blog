package comments

import (
	"blog-api/internal/posts"
	"blog-api/internal/user"
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

func clampLimit(limit int64) int64 {
	if limit <= 0 {
		return 10
	} else if limit > 20 {
		return 20
	}
	return limit
}

type Repository interface {
	Create(ctx context.Context, comment Comment) (Comment, error)
	FindByID(ctx context.Context, id primitive.ObjectID) (Comment, error)
	ListByPostID(ctx context.Context, postID primitive.ObjectID, nextCursor string, limit int64) ([]Comment, error)
	Delete(ctx context.Context, id primitive.ObjectID) error
}

type PostRepo interface {
	GetByID(ctx context.Context, postID primitive.ObjectID) (posts.Post, error)
	IncrementCommentsCount(ctx context.Context, postID primitive.ObjectID, delta int64) error
}

type UserRepo interface {
	FinduserByID(ctx context.Context, id string) (user.User, error)
}

type Service struct {
	repo     Repository
	postRepo PostRepo
	userRepo UserRepo
}

func NewService(repo Repository, postRepo PostRepo, userRepo UserRepo) *Service {
	return &Service{
		repo:     repo,
		postRepo: postRepo,
		userRepo: userRepo,
	}
}

func (s *Service) AddComment(ctx context.Context, postIDStr string, authorIDStr string, input CreateCommentRequest) (Comment, error) {
	postID, err := primitive.ObjectIDFromHex(postIDStr)
	if err != nil {
		return Comment{}, ErrInvalidID
	}

	authorID, err := primitive.ObjectIDFromHex(authorIDStr)
	if err != nil {
		return Comment{}, ErrInvalidID
	}

	content := strings.TrimSpace(input.Content)
	if content == "" {
		return Comment{}, fmt.Errorf("%w: content cannot be empty", ErrInvalidInput)
	}

	// 1. Verify post exists
	post, err := s.postRepo.GetByID(ctx, postID)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return Comment{}, ErrPostNotFound
		}
		return Comment{}, err
	}

	// Only published posts can receive comments
	if post.Status != posts.StatusPublished {
		return Comment{}, ErrPostNotFound
	}

	// 2. Fetch author details
	u, err := s.userRepo.FinduserByID(ctx, authorIDStr)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return Comment{}, user.ErrUserNotFound
		}
		return Comment{}, err
	}

	authorName := strings.TrimSpace(u.FirstName + " " + u.LastName)
	if authorName == "" {
		authorName = u.Email
	}

	// 3. Verify parent comment if nested reply
	var parentID *primitive.ObjectID
	if input.ParentID != nil && strings.TrimSpace(*input.ParentID) != "" {
		pID, err := primitive.ObjectIDFromHex(*input.ParentID)
		if err != nil {
			return Comment{}, ErrInvalidID
		}

		parent, err := s.repo.FindByID(ctx, pID)
		if err != nil {
			if errors.Is(err, mongo.ErrNoDocuments) {
				return Comment{}, ErrParentNotFound
			}
			return Comment{}, err
		}

		if parent.PostID != postID {
			return Comment{}, fmt.Errorf("%w: parent comment does not belong to this post", ErrInvalidInput)
		}

		parentID = &pID
	}

	now := time.Now()
	newComment := Comment{
		PostID:     postID,
		AuthorID:   authorID,
		AuthorName: authorName,
		ParentID:   parentID,
		Content:    content,
		CreatedAt:  now,
		UpdatedAt:  now,
	}

	created, err := s.repo.Create(ctx, newComment)
	if err != nil {
		return Comment{}, err
	}

	// Increment comments count on post (best effort)
	_ = s.postRepo.IncrementCommentsCount(ctx, postID, 1)

	return created, nil
}

func (s *Service) ListComments(ctx context.Context, postIDStr string, cursor string, limit int64) ([]Comment, PaginationMeta, error) {
	postID, err := primitive.ObjectIDFromHex(postIDStr)
	if err != nil {
		return nil, PaginationMeta{}, ErrInvalidID
	}

	limit = clampLimit(limit)

	comments, err := s.repo.ListByPostID(ctx, postID, cursor, limit)
	if err != nil {
		return nil, PaginationMeta{}, err
	}

	hasNext := int64(len(comments)) == limit
	var nextCursor string
	if hasNext && len(comments) > 0 {
		nextCursor = comments[len(comments)-1].ID.Hex()
	}

	meta := PaginationMeta{
		Limit:      limit,
		HasNext:    hasNext,
		NextCursor: nextCursor,
		Count:      int64(len(comments)),
	}

	return comments, meta, nil
}

func (s *Service) DeleteComment(ctx context.Context, commentIDStr string, requesterIDStr string, requesterRole string) error {
	commentID, err := primitive.ObjectIDFromHex(commentIDStr)
	if err != nil {
		return ErrInvalidID
	}

	requesterID, err := primitive.ObjectIDFromHex(requesterIDStr)
	if err != nil {
		return ErrInvalidID
	}

	comment, err := s.repo.FindByID(ctx, commentID)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return ErrNotFound
		}
		return err
	}

	// Admin can delete any comment
	if requesterRole == user.RoleAdmin {
		return s.performDelete(ctx, comment)
	}

	// Comment author can delete own comment
	if comment.AuthorID == requesterID {
		return s.performDelete(ctx, comment)
	}

	// Post author can delete comments on their own post
	post, err := s.postRepo.GetByID(ctx, comment.PostID)
	if err == nil && post.AuthorID == requesterID {
		return s.performDelete(ctx, comment)
	}

	return ErrForbidden
}

func (s *Service) performDelete(ctx context.Context, comment Comment) error {
	if err := s.repo.Delete(ctx, comment.ID); err != nil {
		return err
	}

	_ = s.postRepo.IncrementCommentsCount(ctx, comment.PostID, -1)
	return nil
}
