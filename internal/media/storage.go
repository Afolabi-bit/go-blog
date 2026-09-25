package media

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

// Storage defines the interface for persisting uploaded media assets.
type Storage interface {
	Save(ctx context.Context, filename string, r io.Reader, size int64, mimeType string) (string, error)
}

// LocalStorage persists files directly to the local filesystem.
type LocalStorage struct {
	UploadDir string
}

// NewLocalStorage creates a new LocalStorage instance pointing to uploadDir.
func NewLocalStorage(uploadDir string) *LocalStorage {
	if uploadDir == "" {
		uploadDir = "./uploads"
	}
	return &LocalStorage{UploadDir: uploadDir}
}

// Save writes the uploaded reader content to the local filesystem and returns the relative URL.
func (l *LocalStorage) Save(ctx context.Context, filename string, r io.Reader, size int64, mimeType string) (string, error) {
	if err := os.MkdirAll(l.UploadDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create upload directory: %w", err)
	}

	destPath := filepath.Join(l.UploadDir, filename)
	destFile, err := os.Create(destPath)
	if err != nil {
		return "", fmt.Errorf("failed to create destination file: %w", err)
	}
	defer destFile.Close()

	if _, err := io.Copy(destFile, r); err != nil {
		return "", fmt.Errorf("failed to write file content: %w", err)
	}

	return "/uploads/" + filename, nil
}
