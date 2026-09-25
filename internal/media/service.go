package media

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
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
	UploadDir   string
	MaxFileSize int64
}

func NewService(uploadDir string) *Service {
	if uploadDir == "" {
		uploadDir = "./uploads"
	}
	return &Service{
		UploadDir:   uploadDir,
		MaxFileSize: DefaultMaxFileSize,
	}
}

func (s *Service) SaveFile(file multipart.File, header *multipart.FileHeader) (UploadResponse, error) {
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

	// Ensure upload directory exists
	if err := os.MkdirAll(s.UploadDir, 0755); err != nil {
		return UploadResponse{}, fmt.Errorf("failed to create upload directory: %w", err)
	}

	// Generate safe, unique filename
	randomBytes := make([]byte, 8)
	_, _ = rand.Read(randomBytes)
	uniqueName := fmt.Sprintf("%d-%s%s", time.Now().Unix(), hex.EncodeToString(randomBytes), ext)

	destPath := filepath.Join(s.UploadDir, uniqueName)
	destFile, err := os.Create(destPath)
	if err != nil {
		return UploadResponse{}, fmt.Errorf("failed to create destination file: %w", err)
	}
	defer destFile.Close()

	written, err := io.Copy(destFile, file)
	if err != nil {
		return UploadResponse{}, fmt.Errorf("failed to write file content: %w", err)
	}

	return UploadResponse{
		URL:      "/uploads/" + uniqueName,
		Filename: uniqueName,
		Size:     written,
		MimeType: detectedMime,
	}, nil
}
