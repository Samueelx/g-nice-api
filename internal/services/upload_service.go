package services

import (
	"context"
	"errors"
	"fmt"
	"mime/multipart"
	"path/filepath"
	"strings"

	"github.com/Samueelx/g-nice-api/internal/storage"
	"github.com/google/uuid"
)

// ── Constants ─────────────────────────────────────────────────────────────────

const maxUploadBytes = 10 << 20 // 10 MB

// allowedMIMETypes maps permitted MIME types to their canonical file extension.
var allowedMIMETypes = map[string]string{
	"image/jpeg": ".jpg",
	"image/png":  ".png",
	"image/gif":  ".gif",
	"image/webp": ".webp",
	"video/mp4":  ".mp4",
}

// ── Sentinel errors ───────────────────────────────────────────────────────────

var (
	ErrFileTooLarge          = errors.New("file exceeds the 10 MB size limit")
	ErrUnsupportedMediaType  = errors.New("unsupported file type; allowed: jpeg, png, gif, webp, mp4")
)

// ── Interface ─────────────────────────────────────────────────────────────────

// UploadService handles media upload validation and storage.
type UploadService interface {
	// UploadMedia validates the file, generates a unique S3 key under prefix,
	// streams it to object storage, and returns the public URL.
	// prefix should be "posts" or "avatars".
	UploadMedia(ctx context.Context, userID uint, prefix string, file multipart.File, header *multipart.FileHeader) (string, error)
}

// ── Implementation ────────────────────────────────────────────────────────────

type uploadService struct {
	store storage.Storage
}

// NewUploadService constructs an UploadService backed by the given Storage.
func NewUploadService(store storage.Storage) UploadService {
	return &uploadService{store: store}
}

// UploadMedia validates then streams the file to object storage.
func (s *uploadService) UploadMedia(
	ctx context.Context,
	userID uint,
	prefix string,
	file multipart.File,
	header *multipart.FileHeader,
) (string, error) {
	// 1. Enforce size ceiling.
	if header.Size > maxUploadBytes {
		return "", ErrFileTooLarge
	}

	// 2. Detect MIME type from the Content-Type header sent by the client.
	//    Gin already reads this from the multipart part headers.
	contentType := header.Header.Get("Content-Type")
	// Normalise: strip any ";charset=..." suffix.
	if idx := strings.Index(contentType, ";"); idx != -1 {
		contentType = strings.TrimSpace(contentType[:idx])
	}

	ext, ok := allowedMIMETypes[contentType]
	if !ok {
		// Fall back to the file extension the client provided.
		ext = strings.ToLower(filepath.Ext(header.Filename))
		found := false
		for _, allowedExt := range allowedMIMETypes {
			if ext == allowedExt {
				found = true
				break
			}
		}
		if !found {
			return "", ErrUnsupportedMediaType
		}
		// Re-derive content type from the extension for the S3 header.
		for mime, e := range allowedMIMETypes {
			if e == ext {
				contentType = mime
				break
			}
		}
	}

	// 3. Build a unique, collision-proof object key.
	//    Pattern: {prefix}/{userID}/{uuid}{ext}
	//    e.g.    posts/42/a3f21c8d-...jpg
	key := fmt.Sprintf("%s/%d/%s%s", prefix, userID, uuid.New().String(), ext)

	// 4. Stream to storage.
	result, err := s.store.Upload(ctx, key, file, contentType)
	if err != nil {
		return "", fmt.Errorf("upload: storage error: %w", err)
	}

	return result.URL, nil
}
