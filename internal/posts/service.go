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

func clampLimit(limit int64) int64 {
	if limit <= 0 {
		return 10
	} else if limit > 20 {
		return 20
	}
	return limit
}

var (
	ErrNotFound     = errors.New("post not found")
	ErrForbidden    = errors.New("permission denied")
	ErrInvalidInput = errors.New("invalid input")
	ErrInvalidID    = errors.New("invalid post id format")
)

type Repository interface {
	CreatePost(ctx context.Context, post Post) (Post, error)
	GetByID(ctx context.Context, postID primitive.ObjectID) (Post, error)
	GetBySlug(ctx context.Context, slug string) (Post, error)
	ExistsBySlug(ctx context.Context, slug string) (bool, error)
	ListPublished(ctx context.Context, nextCursor string, limit int64, filter PostFilter) ([]Post, error)
	ListByAuthor(ctx context.Context, authorID primitive.ObjectID, nextCursor string, limit int64) ([]Post, error)
	ListAllAdmin(ctx context.Context, nextCursor string, maxLimit int64, filter PostFilter) ([]Post, error)
	Update(ctx context.Context, postID primitive.ObjectID, authorID *primitive.ObjectID, update UpdatePostRequest) (Post, error)
	Delete(ctx context.Context, postID primitive.ObjectID, authorID *primitive.ObjectID) error
	IncrementCommentsCount(ctx context.Context, postID primitive.ObjectID, delta int64) error
	IncrementLikesCount(ctx context.Context, postID primitive.ObjectID, delta int64) error
	CountPublishedByAuthor(ctx context.Context, authorID primitive.ObjectID) (int64, error)
	GetTags(ctx context.Context) ([]TagItem, error)
	GetAuthorStats(ctx context.Context, authorID primitive.ObjectID) (AuthorStats, error)
	SetFeatured(ctx context.Context, postID primitive.ObjectID, isFeatured bool) (Post, error)
	GetFeaturedPost(ctx context.Context) (Post, error)
}

type UserRepo interface {
	FinduserByID(ctx context.Context, id string) (user.User, error)
}

type LikesRepo interface {
	FindLikedPostIDs(ctx context.Context, postIDs []primitive.ObjectID, userID primitive.ObjectID) (map[primitive.ObjectID]bool, error)
	FindLike(ctx context.Context, postID, userID primitive.ObjectID) (bool, error)
}

type Service struct {
	repo  Repository
	user  UserRepo
	likes LikesRepo
}

func NewService(repo Repository, user UserRepo) *Service {
	return &Service{
		repo: repo,
		user: user,
	}
}

func (s *Service) WithLikesRepo(likes LikesRepo) *Service {
	s.likes = likes
	return s
}

func generateSlug(title string) string {
	title = strings.ToLower(strings.TrimSpace(title))

	slug := slugRegex.ReplaceAllString(title, "-")

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
		return Post{}, fmt.Errorf("%w: title and content are required", ErrInvalidInput)
	}

	if strings.TrimSpace(input.Status) == "" {
		input.Status = StatusDraft
	} else if strings.TrimSpace(input.Status) != StatusDraft && strings.TrimSpace(input.Status) != StatusPublished {
		return Post{}, fmt.Errorf("%w: status must be '%s' or '%s'", ErrInvalidInput, StatusDraft, StatusPublished)
	}

	var sanitizedTags []string

	for _, tag := range input.Tags {
		clean := strings.TrimSpace(tag)

		if clean != "" {
			sanitizedTags = append(sanitizedTags, clean)
		}
	}

	input.Tags = sanitizedTags

	baseSlug := generateSlug(title)
	slug := baseSlug
	counter := 1
	for {
		exists, err := s.repo.ExistsBySlug(ctx, slug)
		if err != nil {
			return Post{}, err
		}
		if !exists {
			break
		}
		slug = fmt.Sprintf("%s-%d", baseSlug, counter)
		counter++
	}

	now := time.Now()
	newPost := Post{
		AuthorID:   authorID,
		AuthorName: authorName,
		Title:      title,
		Slug:       slug,
		Content:    content,
		CoverImage: strings.TrimSpace(input.CoverImage),
		Status:     input.Status,
		Tags:       input.Tags,
		CreatedAt:  now,
		UpdatedAt:  now,
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
		return Post{}, ErrInvalidID
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
			post.ReadTime = CalculateReadTime(post.Content)
			return post, nil
		}

		return Post{}, ErrForbidden
	}

	post.ReadTime = CalculateReadTime(post.Content)
	if requesterID != nil && s.likes != nil {
		if userObjID, err := primitive.ObjectIDFromHex(*requesterID); err == nil {
			post.LikedByMe, _ = s.likes.FindLike(ctx, post.ID, userObjID)
		}
	}

	return post, nil
}

func (s *Service) GetPostBySlug(ctx context.Context, slug string, requesterID *string, requesterRole *string) (Post, error) {
	cleanSlug := strings.TrimSpace(slug)
	if cleanSlug == "" {
		return Post{}, ErrNotFound
	}

	post, err := s.repo.GetBySlug(ctx, cleanSlug)
	if err != nil || errors.Is(err, mongo.ErrNoDocuments) {
		return Post{}, ErrNotFound
	}

	if post.Status != StatusPublished {
		if requesterID == nil || requesterRole == nil {
			return Post{}, ErrNotFound
		}

		if *requesterRole == RoleAdmin {
			post.ReadTime = CalculateReadTime(post.Content)
			return post, nil
		}

		requesterObjID, err := primitive.ObjectIDFromHex(*requesterID)
		if err == nil && requesterObjID == post.AuthorID {
			post.ReadTime = CalculateReadTime(post.Content)
			return post, nil
		}

		return Post{}, ErrForbidden
	}

	post.ReadTime = CalculateReadTime(post.Content)
	if requesterID != nil && s.likes != nil {
		if userObjID, err := primitive.ObjectIDFromHex(*requesterID); err == nil {
			post.LikedByMe, _ = s.likes.FindLike(ctx, post.ID, userObjID)
		}
	}

	return post, nil
}

func (s *Service) ListPublicPosts(ctx context.Context, nextCursor string, limit int64, filter PostFilter, requesterID *string) ([]PostListItem, PaginationMeta, error) {
	limit = clampLimit(limit)

	posts, err := s.repo.ListPublished(ctx, nextCursor, limit, filter)
	if err != nil {
		return []PostListItem{}, PaginationMeta{}, err
	}

	var likedMap map[primitive.ObjectID]bool
	if requesterID != nil && s.likes != nil && len(posts) > 0 {
		if userObjID, err := primitive.ObjectIDFromHex(*requesterID); err == nil {
			postIDs := make([]primitive.ObjectID, len(posts))
			for i, p := range posts {
				postIDs[i] = p.ID
			}
			likedMap, _ = s.likes.FindLikedPostIDs(ctx, postIDs, userObjID)
		}
	}

	items := make([]PostListItem, len(posts))
	for i, p := range posts {
		liked := false
		if likedMap != nil {
			liked = likedMap[p.ID]
		}
		items[i] = ToPostListItem(p, liked)
	}

	var nextCursorStr string
	hasNext := int64(len(posts)) == limit

	if hasNext && len(posts) > 0 {
		nextCursorStr = posts[len(posts)-1].ID.Hex()
	}

	pMeta := PaginationMeta{
		Limit:      limit,
		HasNext:    hasNext,
		NextCursor: nextCursorStr,
		Count:      int64(len(items)),
	}

	return items, pMeta, nil
}

func (s *Service) ListMyPosts(ctx context.Context, authorID primitive.ObjectID, nextCursor string, limit int64) ([]PostListItem, PaginationMeta, error) {
	limit = clampLimit(limit)

	posts, err := s.repo.ListByAuthor(ctx, authorID, nextCursor, limit)
	if err != nil {
		return []PostListItem{}, PaginationMeta{}, err
	}

	items := make([]PostListItem, len(posts))
	for i, p := range posts {
		items[i] = ToPostListItem(p, false)
	}

	hasNext := int64(len(posts)) == limit

	var nextCursorStr string
	if hasNext && len(posts) > 0 {
		nextCursorStr = posts[len(posts)-1].ID.Hex()
	}

	pMeta := PaginationMeta{
		Limit:      limit,
		HasNext:    hasNext,
		NextCursor: nextCursorStr,
		Count:      int64(len(items)),
	}

	return items, pMeta, nil
}

func (s *Service) ListAllAdmin(ctx context.Context, nextCursor string, limit int64, filter PostFilter) ([]PostListItem, PaginationMeta, error) {
	limit = clampLimit(limit)

	posts, err := s.repo.ListAllAdmin(ctx, nextCursor, limit, filter)
	if err != nil {
		return []PostListItem{}, PaginationMeta{}, err
	}

	items := make([]PostListItem, len(posts))
	for i, p := range posts {
		items[i] = ToPostListItem(p, false)
	}

	hasNext := int64(len(posts)) == limit

	var nextCursorStr string
	if hasNext && len(posts) > 0 {
		nextCursorStr = posts[len(posts)-1].ID.Hex()
	}

	pMeta := PaginationMeta{
		Limit:      limit,
		HasNext:    hasNext,
		NextCursor: nextCursorStr,
		Count:      int64(len(items)),
	}

	return items, pMeta, nil
}

func (s *Service) UpdatePost(ctx context.Context, postID string, requesterID primitive.ObjectID, requesterRole string, input UpdatePostRequest) (Post, error) {
	objID, err := primitive.ObjectIDFromHex(postID)

	if err != nil {
		return Post{}, ErrInvalidID
	}

	status := input.Status
	title := input.Title
	content := input.Content

	if status != nil {
		if *status != StatusDraft && *status != StatusPublished {
			return Post{}, fmt.Errorf("%w: status must be '%s' or '%s'", ErrInvalidInput, StatusDraft, StatusPublished)
		}
	}

	if title != nil {
		if strings.EqualFold(strings.TrimSpace(*title), "") {
			return Post{}, fmt.Errorf("%w: title cannot be empty", ErrInvalidInput)
		}
	}

	if content != nil {
		if strings.EqualFold(strings.TrimSpace(*content), "") {
			return Post{}, fmt.Errorf("%w: content cannot be empty", ErrInvalidInput)
		}
	}

	var authorID *primitive.ObjectID
	switch requesterRole {
	case RoleAdmin:
		authorID = nil
	case RoleAuthor:
		authorID = &requesterID
	default:
		return Post{}, ErrForbidden
	}

	post, err := s.repo.Update(ctx, objID, authorID, input)

	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return Post{}, ErrNotFound
		}
		return Post{}, err
	}

	return post, nil
}

func (s *Service) DeletePost(ctx context.Context, postID string, requesterID primitive.ObjectID, requesterRole string) error {
	objID, err := primitive.ObjectIDFromHex(postID)

	if err != nil {
		return ErrInvalidID
	}

	var authorID *primitive.ObjectID
	switch requesterRole {
	case RoleAdmin:
		authorID = nil
	case RoleAuthor:
		authorID = &requesterID
	default:
		return ErrForbidden
	}

	err = s.repo.Delete(ctx, objID, authorID)

	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return ErrNotFound
		}
		return err
	}

	return nil
}

func (s *Service) GetAuthorProfile(ctx context.Context, authorID string) (AuthorProfile, error) {
	objID, err := primitive.ObjectIDFromHex(authorID)
	if err != nil {
		return AuthorProfile{}, ErrInvalidID
	}

	u, err := s.user.FinduserByID(ctx, authorID)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return AuthorProfile{}, ErrNotFound
		}
		return AuthorProfile{}, err
	}

	totalPosts, err := s.repo.CountPublishedByAuthor(ctx, objID)
	if err != nil {
		totalPosts = 0
	}

	return AuthorProfile{
		ID:         u.ID.Hex(),
		FullName:   u.FirstName + " " + u.LastName,
		Bio:        u.Bio,
		AvatarURL:  u.AvatarURL,
		Role:       u.Role,
		TotalPosts: totalPosts,
		CreatedAt:  u.CreatedAt,
	}, nil
}

func (s *Service) GetTags(ctx context.Context) ([]TagItem, error) {
	return s.repo.GetTags(ctx)
}

func (s *Service) GetAuthorStats(ctx context.Context, authorID primitive.ObjectID) (AuthorStats, error) {
	return s.repo.GetAuthorStats(ctx, authorID)
}

func (s *Service) SetFeatured(ctx context.Context, postID string, isFeatured bool) (Post, error) {
	objID, err := primitive.ObjectIDFromHex(postID)
	if err != nil {
		return Post{}, ErrInvalidID
	}
	return s.repo.SetFeatured(ctx, objID, isFeatured)
}

func (s *Service) GetFeaturedPost(ctx context.Context) (Post, error) {
	post, err := s.repo.GetFeaturedPost(ctx)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return Post{}, ErrNotFound
		}
		return Post{}, err
	}
	post.ReadTime = CalculateReadTime(post.Content)
	return post, nil
}

