package media

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"strings"
	"time"
)

var allowedMimeTypes = map[string]string{
	"image/jpeg": ".jpg",
	"image/png":  ".png",
	"image/webp": ".webp",
	"image/gif":  ".gif",
}

const DefaultMaxFileSize int64 = 5 * 1024 * 1024 // 5MB

type Service struct {
	storage     Storage
	MaxFileSize int64
}

// NewService creates a media service backed by the provided Storage provider.
func NewService(storage Storage) *Service {
	if storage == nil {
		storage = NewLocalStorage("./uploads")
	}
	return &Service{
		storage:     storage,
		MaxFileSize: DefaultMaxFileSize,
	}
}

// NewLocalService creates a media service backed by local filesystem storage.
func NewLocalService(uploadDir string) *Service {
	return NewService(NewLocalStorage(uploadDir))
}

// SaveFile validates file size, sniffs MIME type, and saves through the configured storage provider.
func (s *Service) SaveFile(ctx context.Context, file multipart.File, header *multipart.FileHeader) (UploadResponse, error) {
	if header.Size > s.MaxFileSize {
		return UploadResponse{}, ErrFileTooLarge
	}

	// Read first 512 bytes to sniff actual MIME type
	buf := make([]byte, 512)
	n, err := file.Read(buf)
	if err != nil && err != io.EOF {
		return UploadResponse{}, fmt.Errorf("failed to read file header: %w", err)
	}

	detectedMime := http.DetectContentType(buf[:n])
	// Strip parameters like charset if present
	if idx := strings.Index(detectedMime, ";"); idx != -1 {
		detectedMime = strings.TrimSpace(detectedMime[:idx])
	}

	ext, allowed := allowedMimeTypes[detectedMime]
	if !allowed {
		return UploadResponse{}, ErrInvalidFileType
	}

	// Rewind file pointer
	if _, err := file.Seek(0, io.SeekStart); err != nil {
		return UploadResponse{}, fmt.Errorf("failed to seek file: %w", err)
	}

	// Generate safe, unique filename
	randomBytes := make([]byte, 8)
	_, _ = rand.Read(randomBytes)
	uniqueName := fmt.Sprintf("%d-%s%s", time.Now().Unix(), hex.EncodeToString(randomBytes), ext)

	url, err := s.storage.Save(ctx, uniqueName, file, header.Size, detectedMime)
	if err != nil {
		return UploadResponse{}, err
	}

	return UploadResponse{
		URL:      url,
		Filename: uniqueName,
		Size:     header.Size,
		MimeType: detectedMime,
	}, nil
}
