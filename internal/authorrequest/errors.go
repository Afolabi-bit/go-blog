package authorrequest

import "errors"

var (
	ErrNotFound         = errors.New("author request not found")
	ErrPendingExists    = errors.New("a pending author request already exists")
	ErrAlreadyAuthor    = errors.New("user is already an author or administrator")
	ErrInvalidInput     = errors.New("invalid input")
	ErrInvalidID        = errors.New("invalid author request id format")
	ErrAlreadyProcessed = errors.New("author request has already been reviewed")
	ErrForbidden        = errors.New("permission denied")
)
