package user

import (
	"blog-api/internal/auth"
	"context"
	"errors"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"golang.org/x/crypto/bcrypt"
)

type SessionRepository interface {
	StoreToken(ctx context.Context, userID primitive.ObjectID, rawToken string, ttl time.Duration) error
	ValidateAndRotate(ctx context.Context, rawToken string, newRawToken string, newTTL time.Duration) (primitive.ObjectID, error)
	RevokeToken(ctx context.Context, rawToken string) error
	RevokeAllForUser(ctx context.Context, userID primitive.ObjectID) error
}

type UserRepository interface {
	FindUserByEmail(ctx context.Context, email string) (User, error)
	FinduserByID(ctx context.Context, id string) (User, error)
	CreateUser(ctx context.Context, user User) (User, error)
	UpdateRole(ctx context.Context, userID primitive.ObjectID, newRole string) error
	UpdateProfile(ctx context.Context, userID primitive.ObjectID, req UpdateProfileRequest) (User, error)
	UpdatePassword(ctx context.Context, userID primitive.ObjectID, newPasswordHash string) error
}

type Service struct {
	repo        UserRepository
	sessionRepo SessionRepository
	jwtSecret   string
}

func NewService(repo UserRepository, sessionRepo SessionRepository, jwtSecret string) *Service {
	return &Service{
		repo:        repo,
		sessionRepo: sessionRepo,
		jwtSecret:   jwtSecret,
	}
}

func (s *Service) Register(ctx context.Context, input RegisterRequest) (AuthResponse, error) {
	email := input.Email
	password := input.Password
	firstName := input.FirstName
	lastName := input.LastName

	if email == "" {
		return AuthResponse{}, fmt.Errorf("%w: email is required", ErrInvalidInput)
	}
	if firstName == "" {
		return AuthResponse{}, fmt.Errorf("%w: first name is required", ErrInvalidInput)
	}
	if lastName == "" {
		return AuthResponse{}, fmt.Errorf("%w: last name is required", ErrInvalidInput)
	}
	if password == "" {
		return AuthResponse{}, fmt.Errorf("%w: password is required", ErrInvalidInput)
	}
	if len(password) < 6 {
		return AuthResponse{}, fmt.Errorf("%w: password must be at least 6 characters", ErrInvalidInput)
	}

	_, err := s.repo.FindUserByEmail(ctx, email)

	if err == nil {
		return AuthResponse{}, ErrEmailAlreadyExists
	}

	if !errors.Is(err, mongo.ErrNoDocuments) {
		return AuthResponse{}, err
	}

	hashedBytes, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)

	if err != nil {
		return AuthResponse{}, fmt.Errorf("failed to hash password: %w", err)
	}

	now := time.Now()

	// All registrations default strictly to reader role for security.
	// Users can apply to become authors through the author-request process.
	role := RoleReader

	user := User{
		Email:        email,
		Role:         role,
		FirstName:    firstName,
		LastName:     lastName,
		PasswordHash: string(hashedBytes),
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	createdUser, err := s.repo.CreateUser(ctx, user)

	if err != nil {
		return AuthResponse{}, err
	}

	token, err := auth.CreateToken(s.jwtSecret, createdUser.ID.Hex(), createdUser.Email, createdUser.Role)
	if err != nil {
		return AuthResponse{}, err
	}

	refreshToken, err := auth.GenerateRefreshToken()
	if err == nil && s.sessionRepo != nil {
		_ = s.sessionRepo.StoreToken(ctx, createdUser.ID, refreshToken, 7*24*time.Hour)
	}

	return AuthResponse{
		Token:        token,
		RefreshToken: refreshToken,
		User:         ToPublic(createdUser),
	}, nil
}

func (s *Service) Login(ctx context.Context, input LoginRequest) (AuthResponse, error) {
	email := input.Email
	password := input.Password

	if email == "" {
		return AuthResponse{}, fmt.Errorf("%w: email is required", ErrInvalidInput)
	}

	if password == "" {
		return AuthResponse{}, fmt.Errorf("%w: password is required", ErrInvalidInput)
	}

	if len(password) < 6 {
		return AuthResponse{}, fmt.Errorf("%w: password must be at least 6 characters", ErrInvalidInput)
	}

	user, err := s.repo.FindUserByEmail(ctx, email)

	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return AuthResponse{}, ErrInvalidCredentials
		}
		return AuthResponse{}, err
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
		return AuthResponse{}, ErrInvalidCredentials
	}

	token, err := auth.CreateToken(s.jwtSecret, user.ID.Hex(), user.Email, user.Role)
	if err != nil {
		return AuthResponse{}, err
	}

	refreshToken, err := auth.GenerateRefreshToken()
	if err == nil && s.sessionRepo != nil {
		_ = s.sessionRepo.StoreToken(ctx, user.ID, refreshToken, 7*24*time.Hour)
	}

	return AuthResponse{
		Token:        token,
		RefreshToken: refreshToken,
		User:         ToPublic(user),
	}, nil
}

func (s *Service) RefreshToken(ctx context.Context, rawRefreshToken string) (AuthResponse, error) {
	if s.sessionRepo == nil {
		return AuthResponse{}, auth.ErrInvalidToken
	}

	newRefreshToken, err := auth.GenerateRefreshToken()
	if err != nil {
		return AuthResponse{}, fmt.Errorf("failed to generate refresh token: %w", err)
	}

	userID, err := s.sessionRepo.ValidateAndRotate(ctx, rawRefreshToken, newRefreshToken, 7*24*time.Hour)
	if err != nil {
		return AuthResponse{}, err
	}

	u, err := s.repo.FinduserByID(ctx, userID.Hex())
	if err != nil {
		return AuthResponse{}, err
	}

	token, err := auth.CreateToken(s.jwtSecret, u.ID.Hex(), u.Email, u.Role)
	if err != nil {
		return AuthResponse{}, err
	}

	return AuthResponse{
		Token:        token,
		RefreshToken: newRefreshToken,
		User:         ToPublic(u),
	}, nil
}

func (s *Service) Logout(ctx context.Context, userID string, rawRefreshToken *string) error {
	if s.sessionRepo == nil {
		return nil
	}

	if rawRefreshToken != nil && *rawRefreshToken != "" {
		_ = s.sessionRepo.RevokeToken(ctx, *rawRefreshToken)
	}

	if uID, err := primitive.ObjectIDFromHex(userID); err == nil {
		_ = s.sessionRepo.RevokeAllForUser(ctx, uID)
	}

	return nil
}

func (s *Service) GetProfile(ctx context.Context, userID string) (PublicUser, error) {
	user, err := s.repo.FinduserByID(ctx, userID)

	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return PublicUser{}, ErrUserNotFound
		}
		return PublicUser{}, err
	}

	return ToPublic(user), nil
}

func (s *Service) UpdateProfile(ctx context.Context, userID string, input UpdateProfileRequest) (PublicUser, error) {
	uID, err := primitive.ObjectIDFromHex(userID)
	if err != nil {
		return PublicUser{}, ErrInvalidInput
	}

	updated, err := s.repo.UpdateProfile(ctx, uID, input)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return PublicUser{}, ErrUserNotFound
		}
		return PublicUser{}, err
	}

	return ToPublic(updated), nil
}

func (s *Service) ChangePassword(ctx context.Context, userID string, input ChangePasswordRequest) error {
	uID, err := primitive.ObjectIDFromHex(userID)
	if err != nil {
		return ErrInvalidInput
	}

	if len(input.NewPassword) < 6 {
		return fmt.Errorf("%w: new password must be at least 6 characters", ErrInvalidInput)
	}

	u, err := s.repo.FinduserByID(ctx, userID)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return ErrUserNotFound
		}
		return err
	}

	if err := bcrypt.CompareHashAndPassword([]byte(u.PasswordHash), []byte(input.OldPassword)); err != nil {
		return ErrInvalidCredentials
	}

	hashed, err := bcrypt.GenerateFromPassword([]byte(input.NewPassword), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("failed to hash password: %w", err)
	}

	if err := s.repo.UpdatePassword(ctx, uID, string(hashed)); err != nil {
		return err
	}

	// Revoke sessions on password change for security
	if s.sessionRepo != nil {
		_ = s.sessionRepo.RevokeAllForUser(ctx, uID)
	}

	return nil
}
