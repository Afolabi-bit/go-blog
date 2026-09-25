package authorrequest

import (
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
	Create(ctx context.Context, req AuthorRequest) (AuthorRequest, error)
	FindPendingByUserID(ctx context.Context, userID primitive.ObjectID) (AuthorRequest, error)
	FindLatestByUserID(ctx context.Context, userID primitive.ObjectID) (AuthorRequest, error)
	FindByID(ctx context.Context, id primitive.ObjectID) (AuthorRequest, error)
	List(ctx context.Context, nextCursor string, limit int64, status string) ([]AuthorRequest, error)
	UpdateReview(ctx context.Context, id primitive.ObjectID, reviewerID primitive.ObjectID, status string, notes string) (AuthorRequest, error)
}

type UserRepo interface {
	FinduserByID(ctx context.Context, id string) (user.User, error)
	UpdateRole(ctx context.Context, userID primitive.ObjectID, newRole string) error
}

type Service struct {
	repo     Repository
	userRepo UserRepo
}

func NewService(repo Repository, userRepo UserRepo) *Service {
	return &Service{
		repo:     repo,
		userRepo: userRepo,
	}
}

func (s *Service) Submit(ctx context.Context, userID string, input SubmitRequest) (AuthorRequest, error) {
	userObjID, err := primitive.ObjectIDFromHex(userID)
	if err != nil {
		return AuthorRequest{}, ErrInvalidID
	}

	u, err := s.userRepo.FinduserByID(ctx, userID)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return AuthorRequest{}, user.ErrUserNotFound
		}
		return AuthorRequest{}, err
	}

	if u.Role != user.RoleReader {
		return AuthorRequest{}, ErrAlreadyAuthor
	}

	existing, err := s.repo.FindPendingByUserID(ctx, userObjID)
	if err == nil && existing.Status == StatusPending {
		return AuthorRequest{}, ErrPendingExists
	}
	if err != nil && !errors.Is(err, mongo.ErrNoDocuments) {
		return AuthorRequest{}, err
	}

	bio := strings.TrimSpace(input.Bio)
	motivation := strings.TrimSpace(input.Motivation)

	if bio == "" || motivation == "" {
		return AuthorRequest{}, fmt.Errorf("%w: bio and motivation are required", ErrInvalidInput)
	}

	var cleanLinks []string
	for _, link := range input.SampleLinks {
		trimmed := strings.TrimSpace(link)
		if trimmed != "" {
			cleanLinks = append(cleanLinks, trimmed)
		}
	}

	now := time.Now()
	newReq := AuthorRequest{
		UserID:      userObjID,
		UserEmail:   u.Email,
		UserName:    u.FirstName + " " + u.LastName,
		Bio:         bio,
		SampleLinks: cleanLinks,
		Motivation:  motivation,
		Status:      StatusPending,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	created, err := s.repo.Create(ctx, newReq)
	if err != nil {
		return AuthorRequest{}, err
	}

	return created, nil
}

func (s *Service) GetMyRequest(ctx context.Context, userID string) (AuthorRequest, error) {
	userObjID, err := primitive.ObjectIDFromHex(userID)
	if err != nil {
		return AuthorRequest{}, ErrInvalidID
	}

	req, err := s.repo.FindLatestByUserID(ctx, userObjID)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return AuthorRequest{}, ErrNotFound
		}
		return AuthorRequest{}, err
	}

	return req, nil
}

func (s *Service) ListRequests(ctx context.Context, cursor string, limit int64, filter Filter) ([]AuthorRequest, PaginationMeta, error) {
	limit = clampLimit(limit)

	requests, err := s.repo.List(ctx, cursor, limit, filter.Status)
	if err != nil {
		return nil, PaginationMeta{}, err
	}

	hasNext := int64(len(requests)) == limit
	var nextCursor string
	if hasNext && len(requests) > 0 {
		nextCursor = requests[len(requests)-1].ID.Hex()
	}

	meta := PaginationMeta{
		Limit:      limit,
		HasNext:    hasNext,
		NextCursor: nextCursor,
		Count:      int64(len(requests)),
	}

	return requests, meta, nil
}

func (s *Service) ReviewRequest(ctx context.Context, requestID string, reviewerID string, input ReviewRequest) (AuthorRequest, error) {
	reqID, err := primitive.ObjectIDFromHex(requestID)
	if err != nil {
		return AuthorRequest{}, ErrInvalidID
	}

	revID, err := primitive.ObjectIDFromHex(reviewerID)
	if err != nil {
		return AuthorRequest{}, ErrInvalidID
	}

	status := strings.ToLower(strings.TrimSpace(input.Status))
	if status != StatusApproved && status != StatusRejected {
		return AuthorRequest{}, fmt.Errorf("%w: status must be '%s' or '%s'", ErrInvalidInput, StatusApproved, StatusRejected)
	}

	existing, err := s.repo.FindByID(ctx, reqID)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return AuthorRequest{}, ErrNotFound
		}
		return AuthorRequest{}, err
	}

	if existing.Status != StatusPending {
		return AuthorRequest{}, ErrAlreadyProcessed
	}

	if status == StatusApproved {
		if err := s.userRepo.UpdateRole(ctx, existing.UserID, user.RoleAuthor); err != nil {
			return AuthorRequest{}, fmt.Errorf("failed to upgrade user role: %w", err)
		}
	}

	updated, err := s.repo.UpdateReview(ctx, reqID, revID, status, input.ReviewNotes)
	if err != nil {
		return AuthorRequest{}, err
	}

	return updated, nil
}
