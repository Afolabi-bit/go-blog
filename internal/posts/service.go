package posts

import (
	"blog-api/internal/user"
	"context"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
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

func (s *Service) CreateNewPost(ctx context.Context, authorID primitive.ObjectID, authorName string, input CreatePostRequest) (Post, error) {
	title := strings.TrimSpace(input.Title)
	content := strings.TrimSpace(input.Content)

	if title == "" || content == "" {
		return Post{}, ErrInvalidInput
	}

	if strings.TrimSpace(input.Status) == "" {
		input.Status = StatusDraft
	} else if strings.TrimSpace(input.Status) != StatusDraft && strings.TrimSpace(input.Status) != StatusPublished {
		return Post{}, ErrInvalidInput
	}

	var sanitizedTags []string

	for _, tag := range input.Tags {
		if trimmed := strings.TrimSpace(tag); trimmed != "" {
			sanitizedTags = append(sanitizedTags, trimmed)
		}
	}

	slug := generateSlug(input.Title)

	newPost := Post{
		Title:      title,
		Content:    content,
		Status:     input.Status,
		Tags:       sanitizedTags,
		AuthorID:   authorID,
		AuthorName: authorName,
		Slug:       slug,
		CreatedAt:  time.Now().UTC(),
		UpdatedAt:  time.Now().UTC(),
	}

	post, err := s.repo.CreatePost(ctx, newPost)

	if err != nil {
		return Post{}, err
	}

	return post, nil
}

func (s *Service) GetPostByID(ctx context.Context, postID string, requesterID *string, requesterRole *string) (Post, error) {
	objID, err := primitive.ObjectIDFromHex(postID)
	if err != nil {
		return Post{}, ErrNotFound
	}

	post, err := s.repo.GetByID(ctx, objID)

	if err != nil || errors.Is(err, mongo.ErrNoDocuments) {
		return Post{}, ErrNotFound
	}

	if post.Status != StatusPublished {

		if requesterID == nil || requesterRole == nil {
			return Post{}, ErrNotFound
		}

		if *requesterRole == RoleAdmin {
			return post, nil
		}

		requesterObjID, err := primitive.ObjectIDFromHex(*requesterID)

		if err == nil && requesterObjID == post.AuthorID {
			return post, nil
		}

		return Post{}, ErrForbidden
	}

	return post, nil
}
