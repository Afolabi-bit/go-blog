package posts

import (
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

const (
	StatusDraft     = "draft"
	StatusPublished = "published"
	RoleAdmin       = "admin"
	RoleAuthor      = "author"
	RoleReader      = "reader"
)

type Post struct {
	ID            primitive.ObjectID `json:"id" bson:"_id,omitempty"`
	AuthorID      primitive.ObjectID `json:"author_id" bson:"author_id,omitempty"`
	AuthorName    string             `json:"author_name" bson:"author_name"`
	Title         string             `json:"title" bson:"title"`
	Slug          string             `json:"slug" bson:"slug"`
	Content       string             `json:"content" bson:"content"`
	CoverImage    string             `json:"cover_image,omitempty" bson:"cover_image,omitempty"`
	Status        string             `json:"status" bson:"status"`
	Tags          []string           `json:"tags" bson:"tags"`
	LikesCount    int64              `json:"likes_count" bson:"likes_count"`
	CommentsCount int64              `json:"comments_count" bson:"comments_count"`
	LikedByMe     bool               `json:"liked_by_me" bson:"-"`
	ReadTime      int                `json:"read_time" bson:"-"`
	IsFeatured    bool               `json:"is_featured" bson:"is_featured"`
	PreviousSlugs []string           `json:"previous_slugs,omitempty" bson:"previous_slugs,omitempty"`
	CreatedAt     time.Time          `json:"created_at" bson:"created_at"`
	UpdatedAt     time.Time          `json:"updated_at" bson:"updated_at"`
}

type PostListItem struct {
	ID            primitive.ObjectID `json:"id" bson:"_id,omitempty"`
	AuthorID      primitive.ObjectID `json:"author_id" bson:"author_id,omitempty"`
	AuthorName    string             `json:"author_name" bson:"author_name"`
	Title         string             `json:"title" bson:"title"`
	Slug          string             `json:"slug" bson:"slug"`
	Excerpt       string             `json:"excerpt" bson:"-"`
	CoverImage    string             `json:"cover_image,omitempty" bson:"cover_image,omitempty"`
	Status        string             `json:"status" bson:"status"`
	Tags          []string           `json:"tags" bson:"tags"`
	ReadTime      int                `json:"read_time" bson:"-"`
	LikesCount    int64              `json:"likes_count" bson:"likes_count"`
	CommentsCount int64              `json:"comments_count" bson:"comments_count"`
	LikedByMe     bool               `json:"liked_by_me" bson:"-"`
	IsFeatured    bool               `json:"is_featured" bson:"is_featured"`
	CreatedAt     time.Time          `json:"created_at" bson:"created_at"`
	UpdatedAt     time.Time          `json:"updated_at" bson:"updated_at"`
}

func ToPostListItem(p Post, likedByMe bool) PostListItem {
	return PostListItem{
		ID:            p.ID,
		AuthorID:      p.AuthorID,
		AuthorName:    p.AuthorName,
		Title:         p.Title,
		Slug:          p.Slug,
		Excerpt:       GenerateExcerpt(p.Content, 180),
		CoverImage:    p.CoverImage,
		Status:        p.Status,
		Tags:          p.Tags,
		ReadTime:      CalculateReadTime(p.Content),
		LikesCount:    p.LikesCount,
		CommentsCount: p.CommentsCount,
		LikedByMe:     likedByMe,
		IsFeatured:    p.IsFeatured,
		CreatedAt:     p.CreatedAt,
		UpdatedAt:     p.UpdatedAt,
	}
}

func GenerateExcerpt(content string, maxLen int) string {
	clean := strings.TrimSpace(content)
	clean = strings.ReplaceAll(clean, "\r\n", " ")
	clean = strings.ReplaceAll(clean, "\n", " ")
	fields := strings.Fields(clean)
	joined := strings.Join(fields, " ")
	if len(joined) <= maxLen {
		return joined
	}
	truncated := joined[:maxLen]
	lastSpace := strings.LastIndex(truncated, " ")
	if lastSpace > 0 {
		truncated = truncated[:lastSpace]
	}
	return truncated + "..."
}

func CalculateReadTime(content string) int {
	words := len(strings.Fields(content))
	minutes := words / 200
	if minutes < 1 {
		return 1
	}
	return minutes
}

type CreatePostRequest struct {
	Title      string   `json:"title"`
	Content    string   `json:"content"`
	CoverImage string   `json:"cover_image,omitempty"`
	Status     string   `json:"status"`
	Tags       []string `json:"tags"`
}

type UpdatePostRequest struct {
	Title      *string   `json:"title" binding:"omitempty"`
	Content    *string   `json:"content" binding:"omitempty"`
	CoverImage *string   `json:"cover_image" binding:"omitempty"`
	Status     *string   `json:"status" binding:"omitempty"`
	Tags       *[]string `json:"tags" binding:"omitempty"`
}

type PostFilter struct {
	AuthorID string `form:"author_id"`
	Status   string `form:"status"`
	Tag      string `form:"tag"`
	Search   string `form:"search"`
	Limit    int64  `form:"limit"`
	Cursor   string `form:"cursor"`
}

type PaginationMeta struct {
	Limit      int64  `json:"limit"`
	HasNext    bool   `json:"has_next"`
	NextCursor string `json:"next_cursor,omitempty"`
	Count      int64  `json:"count,omitempty"`
}

type AuthorProfile struct {
	ID         string    `json:"id"`
	FullName   string    `json:"full_name"`
	Bio        string    `json:"bio,omitempty"`
	AvatarURL  string    `json:"avatar_url,omitempty"`
	Role       string    `json:"role"`
	TotalPosts int64     `json:"total_posts"`
	CreatedAt  time.Time `json:"created_at"`
}

type TagItem struct {
	Name  string `json:"name" bson:"_id"`
	Count int64  `json:"count" bson:"count"`
}

type AuthorStats struct {
	TotalPosts     int64 `json:"total_posts" bson:"total_posts"`
	PublishedPosts int64 `json:"published_posts" bson:"published_posts"`
	DraftPosts     int64 `json:"draft_posts" bson:"draft_posts"`
	TotalLikes     int64 `json:"total_likes" bson:"total_likes"`
	TotalComments  int64 `json:"total_comments" bson:"total_comments"`
}

type SetFeaturedRequest struct {
	IsFeatured bool `json:"is_featured"`
}

