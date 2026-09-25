package comments

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type Comment struct {
	ID         primitive.ObjectID  `json:"id" bson:"_id,omitempty"`
	PostID     primitive.ObjectID  `json:"post_id" bson:"post_id"`
	AuthorID   primitive.ObjectID  `json:"author_id" bson:"author_id"`
	AuthorName string              `json:"author_name" bson:"author_name"`
	ParentID   *primitive.ObjectID `json:"parent_id,omitempty" bson:"parent_id,omitempty"`
	Content    string              `json:"content" bson:"content"`
	CreatedAt  time.Time           `json:"created_at" bson:"created_at"`
	UpdatedAt  time.Time           `json:"updated_at" bson:"updated_at"`
}

type CreateCommentRequest struct {
	Content  string  `json:"content" binding:"required,min=1,max=2000"`
	ParentID *string `json:"parent_id,omitempty" binding:"omitempty"`
}

type CommentFilter struct {
	Cursor string `form:"cursor"`
	Limit  int64  `form:"limit"`
}

type PaginationMeta struct {
	Limit      int64  `json:"limit"`
	HasNext    bool   `json:"has_next"`
	NextCursor string `json:"next_cursor,omitempty"`
	Count      int64  `json:"count,omitempty"`
}
