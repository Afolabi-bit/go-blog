package authorrequest_test

import (
	"blog-api/internal/authorrequest"
	"blog-api/internal/user"
	"context"
	"errors"
	"testing"
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

// mockRepo satisfies authorrequest.Repository
type mockRepo struct {
	createFunc              func(ctx context.Context, req authorrequest.AuthorRequest) (authorrequest.AuthorRequest, error)
	findPendingByUserIDFunc func(ctx context.Context, userID primitive.ObjectID) (authorrequest.AuthorRequest, error)
	findLatestByUserIDFunc  func(ctx context.Context, userID primitive.ObjectID) (authorrequest.AuthorRequest, error)
	findByIDFunc            func(ctx context.Context, id primitive.ObjectID) (authorrequest.AuthorRequest, error)
	listFunc                func(ctx context.Context, nextCursor string, limit int64, status string) ([]authorrequest.AuthorRequest, error)
	updateReviewFunc        func(ctx context.Context, id primitive.ObjectID, reviewerID primitive.ObjectID, status string, notes string) (authorrequest.AuthorRequest, error)
}

func (m *mockRepo) Create(ctx context.Context, req authorrequest.AuthorRequest) (authorrequest.AuthorRequest, error) {
	if m.createFunc != nil {
		return m.createFunc(ctx, req)
	}
	req.ID = primitive.NewObjectID()
	return req, nil
}

func (m *mockRepo) FindPendingByUserID(ctx context.Context, userID primitive.ObjectID) (authorrequest.AuthorRequest, error) {
	if m.findPendingByUserIDFunc != nil {
		return m.findPendingByUserIDFunc(ctx, userID)
	}
	return authorrequest.AuthorRequest{}, mongo.ErrNoDocuments
}

func (m *mockRepo) FindLatestByUserID(ctx context.Context, userID primitive.ObjectID) (authorrequest.AuthorRequest, error) {
	if m.findLatestByUserIDFunc != nil {
		return m.findLatestByUserIDFunc(ctx, userID)
	}
	return authorrequest.AuthorRequest{}, mongo.ErrNoDocuments
}

func (m *mockRepo) FindByID(ctx context.Context, id primitive.ObjectID) (authorrequest.AuthorRequest, error) {
	if m.findByIDFunc != nil {
		return m.findByIDFunc(ctx, id)
	}
	return authorrequest.AuthorRequest{}, mongo.ErrNoDocuments
}

func (m *mockRepo) List(ctx context.Context, nextCursor string, limit int64, status string) ([]authorrequest.AuthorRequest, error) {
	if m.listFunc != nil {
		return m.listFunc(ctx, nextCursor, limit, status)
	}
	return []authorrequest.AuthorRequest{}, nil
}

func (m *mockRepo) UpdateReview(ctx context.Context, id primitive.ObjectID, reviewerID primitive.ObjectID, status string, notes string) (authorrequest.AuthorRequest, error) {
	if m.updateReviewFunc != nil {
		return m.updateReviewFunc(ctx, id, reviewerID, status, notes)
	}
	return authorrequest.AuthorRequest{
		ID:          id,
		Status:      status,
		ReviewedBy:  &reviewerID,
		ReviewNotes: notes,
	}, nil
}

// mockUserRepo satisfies authorrequest.UserRepo
type mockUserRepo struct {
	findUserByIDFunc func(ctx context.Context, id string) (user.User, error)
	updateRoleFunc   func(ctx context.Context, userID primitive.ObjectID, newRole string) error
}

func (m *mockUserRepo) FinduserByID(ctx context.Context, id string) (user.User, error) {
	if m.findUserByIDFunc != nil {
		return m.findUserByIDFunc(ctx, id)
	}
	objID, _ := primitive.ObjectIDFromHex(id)
	return user.User{
		ID:        objID,
		Email:     "reader@example.com",
		FirstName: "John",
		LastName:  "Doe",
		Role:      user.RoleReader,
	}, nil
}

func (m *mockUserRepo) UpdateRole(ctx context.Context, userID primitive.ObjectID, newRole string) error {
	if m.updateRoleFunc != nil {
		return m.updateRoleFunc(ctx, userID, newRole)
	}
	return nil
}

func TestSubmitAuthorRequest_Success(t *testing.T) {
	uID := primitive.NewObjectID()
	userMock := &mockUserRepo{
		findUserByIDFunc: func(ctx context.Context, id string) (user.User, error) {
			return user.User{
				ID:        uID,
				Email:     "test@blog.com",
				FirstName: "Alice",
				LastName:  "Smith",
				Role:      user.RoleReader,
			}, nil
		},
	}

	repoMock := &mockRepo{
		findPendingByUserIDFunc: func(ctx context.Context, userID primitive.ObjectID) (authorrequest.AuthorRequest, error) {
			return authorrequest.AuthorRequest{}, mongo.ErrNoDocuments
		},
		createFunc: func(ctx context.Context, req authorrequest.AuthorRequest) (authorrequest.AuthorRequest, error) {
			req.ID = primitive.NewObjectID()
			return req, nil
		},
	}

	svc := authorrequest.NewService(repoMock, userMock)

	input := authorrequest.SubmitRequest{
		Bio:         "I am a tech writer with 5 years experience.",
		SampleLinks: []string{"https://example.com/post1"},
		Motivation:  "I want to contribute Go articles to the community.",
	}

	result, err := svc.Submit(context.Background(), uID.Hex(), input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.Status != authorrequest.StatusPending {
		t.Errorf("expected status 'pending', got %s", result.Status)
	}
	if result.UserName != "Alice Smith" {
		t.Errorf("expected user name 'Alice Smith', got %s", result.UserName)
	}
}

func TestSubmitAuthorRequest_AlreadyAuthor(t *testing.T) {
	uID := primitive.NewObjectID()
	userMock := &mockUserRepo{
		findUserByIDFunc: func(ctx context.Context, id string) (user.User, error) {
			return user.User{
				ID:   uID,
				Role: user.RoleAuthor,
			}, nil
		},
	}
	repoMock := &mockRepo{}
	svc := authorrequest.NewService(repoMock, userMock)

	input := authorrequest.SubmitRequest{
		Bio:        "Bio",
		Motivation: "Motivation",
	}

	_, err := svc.Submit(context.Background(), uID.Hex(), input)
	if !errors.Is(err, authorrequest.ErrAlreadyAuthor) {
		t.Fatalf("expected ErrAlreadyAuthor, got: %v", err)
	}
}

func TestSubmitAuthorRequest_PendingExists(t *testing.T) {
	uID := primitive.NewObjectID()
	userMock := &mockUserRepo{}
	repoMock := &mockRepo{
		findPendingByUserIDFunc: func(ctx context.Context, userID primitive.ObjectID) (authorrequest.AuthorRequest, error) {
			return authorrequest.AuthorRequest{
				ID:     primitive.NewObjectID(),
				UserID: uID,
				Status: authorrequest.StatusPending,
			}, nil
		},
	}
	svc := authorrequest.NewService(repoMock, userMock)

	input := authorrequest.SubmitRequest{
		Bio:        "Tech enthusiast",
		Motivation: "I want to publish",
	}

	_, err := svc.Submit(context.Background(), uID.Hex(), input)
	if !errors.Is(err, authorrequest.ErrPendingExists) {
		t.Fatalf("expected ErrPendingExists, got: %v", err)
	}
}

func TestReviewAuthorRequest_Approve(t *testing.T) {
	reqID := primitive.NewObjectID()
	applicantID := primitive.NewObjectID()
	reviewerID := primitive.NewObjectID()

	var updatedRole string
	userMock := &mockUserRepo{
		updateRoleFunc: func(ctx context.Context, userID primitive.ObjectID, newRole string) error {
			if userID != applicantID {
				t.Errorf("expected user id %s, got %s", applicantID.Hex(), userID.Hex())
			}
			updatedRole = newRole
			return nil
		},
	}

	repoMock := &mockRepo{
		findByIDFunc: func(ctx context.Context, id primitive.ObjectID) (authorrequest.AuthorRequest, error) {
			return authorrequest.AuthorRequest{
				ID:        reqID,
				UserID:    applicantID,
				Status:    authorrequest.StatusPending,
				CreatedAt: time.Now(),
			}, nil
		},
		updateReviewFunc: func(ctx context.Context, id primitive.ObjectID, revID primitive.ObjectID, status string, notes string) (authorrequest.AuthorRequest, error) {
			return authorrequest.AuthorRequest{
				ID:          id,
				UserID:      applicantID,
				Status:      status,
				ReviewedBy:  &revID,
				ReviewNotes: notes,
			}, nil
		},
	}

	svc := authorrequest.NewService(repoMock, userMock)

	input := authorrequest.ReviewRequest{
		Status:      authorrequest.StatusApproved,
		ReviewNotes: "Welcome as an author!",
	}

	res, err := svc.ReviewRequest(context.Background(), reqID.Hex(), reviewerID.Hex(), input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if res.Status != authorrequest.StatusApproved {
		t.Errorf("expected status approved, got %s", res.Status)
	}

	if updatedRole != user.RoleAuthor {
		t.Errorf("expected updated role 'author', got '%s'", updatedRole)
	}
}

func TestReviewAuthorRequest_Reject(t *testing.T) {
	reqID := primitive.NewObjectID()
	applicantID := primitive.NewObjectID()
	reviewerID := primitive.NewObjectID()

	roleUpdated := false
	userMock := &mockUserRepo{
		updateRoleFunc: func(ctx context.Context, userID primitive.ObjectID, newRole string) error {
			roleUpdated = true
			return nil
		},
	}

	repoMock := &mockRepo{
		findByIDFunc: func(ctx context.Context, id primitive.ObjectID) (authorrequest.AuthorRequest, error) {
			return authorrequest.AuthorRequest{
				ID:     reqID,
				UserID: applicantID,
				Status: authorrequest.StatusPending,
			}, nil
		},
	}

	svc := authorrequest.NewService(repoMock, userMock)

	input := authorrequest.ReviewRequest{
		Status:      authorrequest.StatusRejected,
		ReviewNotes: "Please provide more sample articles before reapplying.",
	}

	res, err := svc.ReviewRequest(context.Background(), reqID.Hex(), reviewerID.Hex(), input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if res.Status != authorrequest.StatusRejected {
		t.Errorf("expected status rejected, got %s", res.Status)
	}

	if roleUpdated {
		t.Errorf("user role should not be updated when request is rejected")
	}
}

func TestReviewAuthorRequest_AlreadyProcessed(t *testing.T) {
	reqID := primitive.NewObjectID()
	reviewerID := primitive.NewObjectID()

	repoMock := &mockRepo{
		findByIDFunc: func(ctx context.Context, id primitive.ObjectID) (authorrequest.AuthorRequest, error) {
			return authorrequest.AuthorRequest{
				ID:     reqID,
				Status: authorrequest.StatusApproved,
			}, nil
		},
	}

	svc := authorrequest.NewService(repoMock, &mockUserRepo{})

	input := authorrequest.ReviewRequest{
		Status: authorrequest.StatusRejected,
	}

	_, err := svc.ReviewRequest(context.Background(), reqID.Hex(), reviewerID.Hex(), input)
	if !errors.Is(err, authorrequest.ErrAlreadyProcessed) {
		t.Fatalf("expected ErrAlreadyProcessed, got: %v", err)
	}
}
