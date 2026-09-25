package media_test

import (
	"blog-api/internal/media"
	"bytes"
	"errors"
	"mime/multipart"
	"os"
	"testing"
)

// Minimal 1x1 valid PNG bytes
var validPNG = []byte{
	0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A,
	0x00, 0x00, 0x00, 0x0D, 0x49, 0x48, 0x44, 0x52,
	0x00, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x01,
	0x08, 0x06, 0x00, 0x00, 0x00, 0x1F, 0x15, 0xC4,
	0x89, 0x00, 0x00, 0x00, 0x0A, 0x49, 0x44, 0x41,
	0x54, 0x78, 0x9C, 0x63, 0x00, 0x01, 0x00, 0x00,
	0x05, 0x00, 0x01, 0x0D, 0x0A, 0x2D, 0xB4, 0x00,
	0x00, 0x00, 0x00, 0x49, 0x45, 0x4E, 0x44, 0xAE,
	0x42, 0x60, 0x82,
}

func createMultipartFile(content []byte, filename string) (multipart.File, *multipart.FileHeader, error) {
	body := new(bytes.Buffer)
	writer := multipart.NewWriter(body)
	part, err := writer.CreateFormFile("file", filename)
	if err != nil {
		return nil, nil, err
	}
	if _, err := part.Write(content); err != nil {
		return nil, nil, err
	}
	if err := writer.Close(); err != nil {
		return nil, nil, err
	}

	reader := multipart.NewReader(body, writer.Boundary())
	form, err := reader.ReadForm(int64(body.Len()))
	if err != nil {
		return nil, nil, err
	}

	files := form.File["file"]
	if len(files) == 0 {
		return nil, nil, errors.New("no files in form")
	}

	file, err := files[0].Open()
	if err != nil {
		return nil, nil, err
	}

	return file, files[0], nil
}

func TestMediaService_SaveFile_Success(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "test_uploads_*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	svc := media.NewService(tempDir)

	file, header, err := createMultipartFile(validPNG, "test.png")
	if err != nil {
		t.Fatalf("failed to create multipart file: %v", err)
	}
	defer file.Close()

	res, err := svc.SaveFile(file, header)
	if err != nil {
		t.Fatalf("unexpected error saving file: %v", err)
	}

	if res.MimeType != "image/png" {
		t.Errorf("expected mime image/png, got %s", res.MimeType)
	}
	if res.Size != int64(len(validPNG)) {
		t.Errorf("expected size %d, got %d", len(validPNG), res.Size)
	}
}

func TestMediaService_SaveFile_InvalidType(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "test_uploads_*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	svc := media.NewService(tempDir)

	textPayload := []byte("<html><body>Not an image</body></html>")
	file, header, err := createMultipartFile(textPayload, "test.html")
	if err != nil {
		t.Fatalf("failed to create multipart file: %v", err)
	}
	defer file.Close()

	_, err = svc.SaveFile(file, header)
	if !errors.Is(err, media.ErrInvalidFileType) {
		t.Fatalf("expected ErrInvalidFileType, got: %v", err)
	}
}

func TestMediaService_SaveFile_TooLarge(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "test_uploads_*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	svc := media.NewService(tempDir)
	svc.MaxFileSize = 10 // small limit for testing

	file, header, err := createMultipartFile(validPNG, "test.png")
	if err != nil {
		t.Fatalf("failed to create multipart file: %v", err)
	}
	defer file.Close()

	_, err = svc.SaveFile(file, header)
	if !errors.Is(err, media.ErrFileTooLarge) {
		t.Fatalf("expected ErrFileTooLarge, got: %v", err)
	}
}
