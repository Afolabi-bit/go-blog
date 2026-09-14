package posts

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

const (
	StatusDraft     = "draft"
	StatusPublished = "published"
)

type Post struct {
	ID         primitive.ObjectID `json:"id" bson:"_id,omitempty"`
	AuthorID   primitive.ObjectID `json:"author_id" bson:"author_id,omitempty"`
	AuthorName string             `json:"author_name" bson:"author_name"`
	Title      string             `json:"title" bson:"title"`
	Slug       string             `json:"slug" bson:"slug"`
	Content    string             `json:"content" bson:"content"`
	Status     string             `json:"status" bson:"status"`
	Tags       []string           `json:"tags" bson:"tags"`
	CreatedAt  time.Time          `json:"created_at" bson:"created_at"`
	UpdatedAt  time.Time          `json:"updated_at" bson:"updated_at"`
}

type CreatePostRequest struct {
	Title   string   `json:"title" binding:"required"`
	Content string   `json:"content" binding:"required"`
	Status  string   `json:"status" binding:"omitempty"`
	Tags    []string `json:"tags" binding:"omitempty"`
}

type UpdatePostRequest struct {
	Title   *string   `json:"title" binding:"omitempty"`
	Content *string   `json:"content" binding:"omitempty"`
	Status  *string   `json:"status" binding:"omitempty"`
	Tags    *[]string `json:"tags" binding:"omitempty"`
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
