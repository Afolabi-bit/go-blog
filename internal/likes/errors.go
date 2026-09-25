package likes

import "errors"

var (
	ErrPostNotFound = errors.New("post not found")
	ErrInvalidID    = errors.New("invalid post or user id format")
)
