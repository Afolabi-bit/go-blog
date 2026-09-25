package user_test

import (
	"blog-api/internal/user"
	"context"
	"errors"
	"testing"
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"golang.org/x/crypto/bcrypt"
)

type mockUserRepo struct {
	findUserByEmailFunc func(ctx context.Context, email string) (user.User, error)
	findUserByIDFunc    func(ctx context.Context, id string) (user.User, error)
	createUserFunc      func(ctx context.Context, u user.User) (user.User, error)
	updateRoleFunc      func(ctx context.Context, userID primitive.ObjectID, newRole string) error
	updateProfileFunc   func(ctx context.Context, userID primitive.ObjectID, req user.UpdateProfileRequest) (user.User, error)
	updatePasswordFunc  func(ctx context.Context, userID primitive.ObjectID, newPasswordHash string) error
}

func (m *mockUserRepo) FindUserByEmail(ctx context.Context, email string) (user.User, error) {
	if m.findUserByEmailFunc != nil {
		return m.findUserByEmailFunc(ctx, email)
	}
	return user.User{}, mongo.ErrNoDocuments
}

func (m *mockUserRepo) FinduserByID(ctx context.Context, id string) (user.User, error) {
	if m.findUserByIDFunc != nil {
		return m.findUserByIDFunc(ctx, id)
	}
	objID, _ := primitive.ObjectIDFromHex(id)
	return user.User{
		ID:        objID,
		Email:     "user@test.com",
		FirstName: "John",
		LastName:  "Doe",
		Role:      user.RoleReader,
	}, nil
}

func (m *mockUserRepo) CreateUser(ctx context.Context, u user.User) (user.User, error) {
	if m.createUserFunc != nil {
		return m.createUserFunc(ctx, u)
	}
	u.ID = primitive.NewObjectID()
	return u, nil
}

func (m *mockUserRepo) UpdateRole(ctx context.Context, userID primitive.ObjectID, newRole string) error {
	if m.updateRoleFunc != nil {
		return m.updateRoleFunc(ctx, userID, newRole)
	}
	return nil
}

func (m *mockUserRepo) UpdateProfile(ctx context.Context, userID primitive.ObjectID, req user.UpdateProfileRequest) (user.User, error) {
	if m.updateProfileFunc != nil {
		return m.updateProfileFunc(ctx, userID, req)
	}
	return user.User{
		ID:        userID,
		Email:     "user@test.com",
		FirstName: "Updated",
		LastName:  "User",
		Bio:       "Updated Bio",
		Role:      user.RoleReader,
	}, nil
}

func (m *mockUserRepo) UpdatePassword(ctx context.Context, userID primitive.ObjectID, newPasswordHash string) error {
	if m.updatePasswordFunc != nil {
		return m.updatePasswordFunc(ctx, userID, newPasswordHash)
	}
	return nil
}

type mockSessionRepo struct {
	storeTokenFunc        func(ctx context.Context, userID primitive.ObjectID, rawToken string, ttl time.Duration) error
	validateAndRotateFunc func(ctx context.Context, rawToken string, newRawToken string, newTTL time.Duration) (primitive.ObjectID, error)
	revokeTokenFunc       func(ctx context.Context, rawToken string) error
	revokeAllForUserFunc  func(ctx context.Context, userID primitive.ObjectID) error
}

func (m *mockSessionRepo) StoreToken(ctx context.Context, userID primitive.ObjectID, rawToken string, ttl time.Duration) error {
	if m.storeTokenFunc != nil {
		return m.storeTokenFunc(ctx, userID, rawToken, ttl)
	}
	return nil
}

func (m *mockSessionRepo) ValidateAndRotate(ctx context.Context, rawToken string, newRawToken string, newTTL time.Duration) (primitive.ObjectID, error) {
	if m.validateAndRotateFunc != nil {
		return m.validateAndRotateFunc(ctx, rawToken, newRawToken, newTTL)
	}
	return primitive.NewObjectID(), nil
}

func (m *mockSessionRepo) RevokeToken(ctx context.Context, rawToken string) error {
	if m.revokeTokenFunc != nil {
		return m.revokeTokenFunc(ctx, rawToken)
	}
	return nil
}

func (m *mockSessionRepo) RevokeAllForUser(ctx context.Context, userID primitive.ObjectID) error {
	if m.revokeAllForUserFunc != nil {
		return m.revokeAllForUserFunc(ctx, userID)
	}
	return nil
}

func TestUpdateProfile_Success(t *testing.T) {
	uID := primitive.NewObjectID()
	userRepo := &mockUserRepo{
		updateProfileFunc: func(ctx context.Context, id primitive.ObjectID, req user.UpdateProfileRequest) (user.User, error) {
			return user.User{
				ID:        id,
				Email:     "test@user.com",
				FirstName: *req.FirstName,
				LastName:  "Smith",
				Bio:       *req.Bio,
			}, nil
		},
	}

	svc := user.NewService(userRepo, &mockSessionRepo{}, "secret123")

	newFirstName := "Jane"
	newBio := "Writer & Researcher"
	input := user.UpdateProfileRequest{
		FirstName: &newFirstName,
		Bio:       &newBio,
	}

	updated, err := svc.UpdateProfile(context.Background(), uID.Hex(), input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if updated.FullName != "Jane Smith" {
		t.Errorf("expected full name 'Jane Smith', got '%s'", updated.FullName)
	}
	if updated.Bio != newBio {
		t.Errorf("expected bio '%s', got '%s'", newBio, updated.Bio)
	}
}

func TestChangePassword_Success(t *testing.T) {
	uID := primitive.NewObjectID()
	initialHash, _ := bcrypt.GenerateFromPassword([]byte("oldpassword123"), bcrypt.DefaultCost)

	passwordUpdated := false
	userRepo := &mockUserRepo{
		findUserByIDFunc: func(ctx context.Context, id string) (user.User, error) {
			return user.User{
				ID:           uID,
				PasswordHash: string(initialHash),
			}, nil
		},
		updatePasswordFunc: func(ctx context.Context, userID primitive.ObjectID, newPasswordHash string) error {
			passwordUpdated = true
			return nil
		},
	}

	sessionsRevoked := false
	sessionRepo := &mockSessionRepo{
		revokeAllForUserFunc: func(ctx context.Context, userID primitive.ObjectID) error {
			sessionsRevoked = true
			return nil
		},
	}

	svc := user.NewService(userRepo, sessionRepo, "secret123")

	input := user.ChangePasswordRequest{
		OldPassword: "oldpassword123",
		NewPassword: "newpassword456",
	}

	err := svc.ChangePassword(context.Background(), uID.Hex(), input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !passwordUpdated {
		t.Errorf("expected password to be updated")
	}
	if !sessionsRevoked {
		t.Errorf("expected sessions to be revoked on password change")
	}
}

func TestChangePassword_WrongOldPassword(t *testing.T) {
	uID := primitive.NewObjectID()
	initialHash, _ := bcrypt.GenerateFromPassword([]byte("correctpassword"), bcrypt.DefaultCost)

	userRepo := &mockUserRepo{
		findUserByIDFunc: func(ctx context.Context, id string) (user.User, error) {
			return user.User{
				ID:           uID,
				PasswordHash: string(initialHash),
			}, nil
		},
	}

	svc := user.NewService(userRepo, &mockSessionRepo{}, "secret123")

	input := user.ChangePasswordRequest{
		OldPassword: "wrongpassword",
		NewPassword: "newpassword456",
	}

	err := svc.ChangePassword(context.Background(), uID.Hex(), input)
	if !errors.Is(err, user.ErrInvalidCredentials) {
		t.Fatalf("expected ErrInvalidCredentials, got: %v", err)
	}
}

func TestRefreshToken_Success(t *testing.T) {
	uID := primitive.NewObjectID()
	sessionRepo := &mockSessionRepo{
		validateAndRotateFunc: func(ctx context.Context, rawToken, newRawToken string, ttl time.Duration) (primitive.ObjectID, error) {
			return uID, nil
		},
	}

	userRepo := &mockUserRepo{
		findUserByIDFunc: func(ctx context.Context, id string) (user.User, error) {
			return user.User{
				ID:    uID,
				Email: "refresh@user.com",
				Role:  user.RoleReader,
			}, nil
		},
	}

	svc := user.NewService(userRepo, sessionRepo, "secret123")

	res, err := svc.RefreshToken(context.Background(), "valid-token")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if res.Token == "" {
		t.Errorf("expected non-empty access token")
	}
	if res.RefreshToken == "" {
		t.Errorf("expected non-empty rotated refresh token")
	}
}
