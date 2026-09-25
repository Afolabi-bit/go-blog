package media

import "errors"

var (
	ErrFileTooLarge    = errors.New("file exceeds maximum allowed size (5MB)")
	ErrInvalidFileType = errors.New("invalid file type; only JPEG, PNG, WebP, and GIF images are allowed")
	ErrNoFileProvided  = errors.New("no file was provided in form data")
)
