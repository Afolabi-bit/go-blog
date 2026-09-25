package posts

import (
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
	CreatedAt     time.Time          `json:"created_at" bson:"created_at"`
	UpdatedAt     time.Time          `json:"updated_at" bson:"updated_at"`
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
	Status string `form:"status"`
	Tag    string `form:"tag"`
	Search string `form:"search"`
	Limit  int64  `form:"limit"`
	Cursor string `form:"cursor"`
}

type PaginationMeta struct {
	Limit      int64  `json:"limit"`
	HasNext    bool   `json:"has_next"`
	NextCursor string `json:"next_cursor,omitempty"`
	Count      int64  `json:"count,omitempty"`
}
