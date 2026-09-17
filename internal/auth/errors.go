package auth

import "errors"

var (
	ErrMissingAuthHeader = errors.New("authorization header required")
	ErrInvalidToken      = errors.New("invalid or expired authorization token")
)
