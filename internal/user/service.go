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
		return AuthResponse{}, fmt.Errorf("Email is required")
	}
	if firstName == "" {
		return AuthResponse{}, fmt.Errorf("First name is required")
	}
	if lastName == "" {
		return AuthResponse{}, fmt.Errorf("Last name is required")
	}
	if password == "" {
		return AuthResponse{}, fmt.Errorf("Password is required")
	}
	if len(password) < 6 {
		return AuthResponse{}, fmt.Errorf("Password must be at least 6 characters")
	}

	_, err := s.repo.FindUserByEmail(ctx, email)

	if err == nil {
		return AuthResponse{}, fmt.Errorf("Email already registered!")
	}

	if !errors.Is(err, mongo.ErrNoDocuments) {
		return AuthResponse{}, err
	}

	hashedBytes, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)

	if err != nil {
		return AuthResponse{}, fmt.Errorf("Failed to hash password: %w", err)
	}

	now := time.Now()

	user := User{
		Email:        email,
		Role:         RoleReader,
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
		return AuthResponse{}, fmt.Errorf("Email is required")
	}

	if password == "" {
		return AuthResponse{}, fmt.Errorf("Password is required")
	}

	if len(password) < 6 {
		return AuthResponse{}, fmt.Errorf("Password must be atleast 6 characters")
	}

	user, err := s.repo.FindUserByEmail(ctx, email)

	if err != nil {
		return AuthResponse{}, err
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
		return AuthResponse{}, errors.New("Invalid Credentials")
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
			return PublicUser{}, errors.New("User not found")
		}
		return PublicUser{}, err
	}

	return ToPublic(user), nil
}
