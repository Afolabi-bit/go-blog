package comments

import "errors"

var (
	ErrNotFound       = errors.New("comment not found")
	ErrPostNotFound   = errors.New("post not found")
	ErrParentNotFound = errors.New("parent comment not found")
	ErrInvalidInput   = errors.New("invalid input")
	ErrInvalidID      = errors.New("invalid comment id format")
	ErrForbidden      = errors.New("permission denied")
)
