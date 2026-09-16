package posts

import (
	"blog-api/internal/user"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"
)

var slugRegex = regexp.MustCompile(`[^a-z0-9]+`)

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

func generateSlug(title string) string {
	title = strings.ToLower(strings.TrimSpace(title))

	reg := regexp.MustCompile(`[^a-z0-9]+`)

	slug := reg.ReplaceAllString(title, "-")

	slug = strings.Trim(slug, "-")

	if slug == "" {
		slug = fmt.Sprintf("p-w-t-%d", time.Now().Unix())
	}

	return slug
}
