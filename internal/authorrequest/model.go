package authorrequest

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

const (
	StatusPending  = "pending"
	StatusApproved = "approved"
	StatusRejected = "rejected"
)

type AuthorRequest struct {
	ID          primitive.ObjectID  `json:"id" bson:"_id,omitempty"`
	UserID      primitive.ObjectID  `json:"user_id" bson:"user_id"`
	UserEmail   string              `json:"user_email" bson:"user_email"`
	UserName    string              `json:"user_name" bson:"user_name"`
	Bio         string              `json:"bio" bson:"bio"`
	SampleLinks []string            `json:"sample_links" bson:"sample_links"`
	Motivation  string              `json:"motivation" bson:"motivation"`
	Status      string              `json:"status" bson:"status"`
	ReviewedBy  *primitive.ObjectID `json:"reviewed_by,omitempty" bson:"reviewed_by,omitempty"`
	ReviewNotes string              `json:"review_notes,omitempty" bson:"review_notes,omitempty"`
	CreatedAt   time.Time           `json:"created_at" bson:"created_at"`
	UpdatedAt   time.Time           `json:"updated_at" bson:"updated_at"`
}

type SubmitRequest struct {
	Bio         string   `json:"bio" binding:"required,min=10,max=1000"`
	SampleLinks []string `json:"sample_links" binding:"omitempty"`
	Motivation  string   `json:"motivation" binding:"required,min=10,max=2000"`
}

type ReviewRequest struct {
	Status      string `json:"status" binding:"required"`
	ReviewNotes string `json:"review_notes" binding:"omitempty,max=500"`
}

type Filter struct {
	Status string `form:"status"`
	Cursor string `form:"cursor"`
	Limit  int64  `form:"limit"`
}

type PaginationMeta struct {
	Limit      int64  `json:"limit"`
	HasNext    bool   `json:"has_next"`
	NextCursor string `json:"next_cursor,omitempty"`
	Count      int64  `json:"count,omitempty"`
}
