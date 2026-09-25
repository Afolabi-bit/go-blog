package user

import (
	"blog-api/internal/auth"
	"context"
	"errors"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/mongo"
	"golang.org/x/crypto/bcrypt"
)

type Service struct {
	repo      *Repo
	jwtSecret string
}

func NewService(repo *Repo, jwtSecret string) *Service {
	return &Service{
		repo:      repo,
		jwtSecret: jwtSecret,
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

	return AuthResponse{
		Token: token,
		User:  ToPublic(createdUser),
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

	return AuthResponse{
		Token: token,
		User:  ToPublic(user),
	}, nil
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
