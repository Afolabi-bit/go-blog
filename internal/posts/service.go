package posts

import (
	"blog-api/internal/user"
	"errors"
)

var (
	ErrNotFound     = errors.New("post not found")
	ErrForbidden    = errors.New("forbidden: you do not have required permission")
	ErrInvalidInput = errors.New("invalid input")
)

type Service struct {
	repo *Repo
	user *user.Repo
}

func NewService(repo *Repo, user *user.Repo) *Service {
	return &Service{
		repo: repo,
		user: user,
	}
}
