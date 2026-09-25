package user

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

const (
	RoleReader = "reader"
	RoleAuthor = "author"
	RoleAdmin  = "admin"
)

type User struct {
	ID           primitive.ObjectID `json:"id" bson:"_id,omitempty"`
	PasswordHash string             `json:"-" bson:"password_hash"`
	Email        string             `json:"email" bson:"email"`
	Role         string             `json:"role" bson:"role"`
	FirstName    string             `json:"first_name" bson:"first_name"`
	LastName     string             `json:"last_name" bson:"last_name"`
	Bio          string             `json:"bio,omitempty" bson:"bio,omitempty"`
	AvatarURL    string             `json:"avatar_url,omitempty" bson:"avatar_url,omitempty"`
	CreatedAt    time.Time          `json:"created_at" bson:"created_at"`
	UpdatedAt    time.Time          `json:"updated_at" bson:"updated_at"`
}

type PublicUser struct {
	ID        string    `json:"id" bson:"_id"`
	FullName  string    `json:"full_name" bson:"-"`
	Email     string    `json:"email" bson:"email"`
	Role      string    `json:"role" bson:"role"`
	Bio       string    `json:"bio,omitempty" bson:"bio,omitempty"`
	AvatarURL string    `json:"avatar_url,omitempty" bson:"avatar_url,omitempty"`
	CreatedAt time.Time `json:"created_at" bson:"created_at"`
	UpdatedAt time.Time `json:"updated_at" bson:"updated_at"`
}

func ToPublic(u User) PublicUser {
	return PublicUser{
		ID:        u.ID.Hex(),
		FullName:  u.FirstName + " " + u.LastName,
		Email:     u.Email,
		Role:      u.Role,
		Bio:       u.Bio,
		AvatarURL: u.AvatarURL,
		CreatedAt: u.CreatedAt,
		UpdatedAt: u.UpdatedAt,
	}
}

type RegisterRequest struct {
	Email     string `json:"email" form:"email" validate:"required,email"`
	Password  string `json:"password" form:"password" validate:"required,min=6"`
	FirstName string `json:"firstName" form:"firstName" validate:"required,min=3"`
	LastName  string `json:"lastName" form:"lastName" validate:"required,min=3"`
	Role      string `json:"role,omitempty" form:"role"`
}

type LoginRequest struct {
	Email    string `json:"email" form:"email" validate:"required,email"`
	Password string `json:"password" form:"password" validate:"required"`
}

type AuthResponse struct {
	Token        string     `json:"token"`
	RefreshToken string     `json:"refresh_token,omitempty"`
	User         PublicUser `json:"user"`
}

type RefreshTokenRequest struct {
	RefreshToken string `json:"refresh_token" binding:"required"`
}

type UpdateProfileRequest struct {
	FirstName *string `json:"first_name" binding:"omitempty,min=2,max=50"`
	LastName  *string `json:"last_name" binding:"omitempty,min=2,max=50"`
	Bio       *string `json:"bio" binding:"omitempty,max=500"`
	AvatarURL *string `json:"avatar_url" binding:"omitempty"`
}

type ChangePasswordRequest struct {
	OldPassword string `json:"old_password" binding:"required"`
	NewPassword string `json:"new_password" binding:"required,min=6"`
}
