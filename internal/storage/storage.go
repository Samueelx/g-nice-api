package storage

import (
	"context"
	"io"
)

// UploadResult holds the outcome of a successful upload.
type UploadResult struct {
	// URL is the publicly accessible HTTPS URL of the uploaded object.
	URL string
	// Key is the S3 object key (e.g. "posts/42/abc123.jpg").
	// Stored alongside the URL so the object can be deleted later if needed.
	Key string
}

// Storage is the interface all storage backends must satisfy.
// Implementations live in this package (e.g. S3Storage, LocalStorage).
type Storage interface {
	// Upload streams r to the backend under the given key with the specified
	// content type and returns the result on success.
	Upload(ctx context.Context, key string, r io.Reader, contentType string) (*UploadResult, error)

	// Delete removes the object identified by key. It is idempotent — deleting
	// a non-existent key must not return an error.
	Delete(ctx context.Context, key string) error
}
