package storage

import (
	"context"
	"fmt"
	"io"

	"cloud.google.com/go/storage"
)

// GCSConfig holds the bucket name for Google Cloud Storage.
// Authentication is handled via Application Default Credentials (ADC),
// which is automatic on Cloud Run and other GCP compute surfaces.
type GCSConfig struct {
	Bucket string
}

// gcsStorage implements Storage backed by Google Cloud Storage.
type gcsStorage struct {
	client *storage.Client
	bucket string
}

// NewGCSStorage constructs a GCS-backed Storage.
// It relies on ADC for authentication — no explicit credentials are needed
// when running on Cloud Run, GKE, or any GCP environment with a service
// account attached.
func NewGCSStorage(cfg GCSConfig) (Storage, error) {
	ctx := context.Background()
	client, err := storage.NewClient(ctx)
	if err != nil {
		return nil, fmt.Errorf("storage: failed to create GCS client: %w", err)
	}

	return &gcsStorage{client: client, bucket: cfg.Bucket}, nil
}

// Upload streams r to GCS at the given key with the given content type.
// The object's Content-Type is set explicitly so browsers handle images and
// videos correctly without content-sniffing.
func (s *gcsStorage) Upload(ctx context.Context, key string, r io.Reader, contentType string) (*UploadResult, error) {
	obj := s.client.Bucket(s.bucket).Object(key)
	w := obj.NewWriter(ctx)
	w.ContentType = contentType

	if _, err := io.Copy(w, r); err != nil {
		// Attempt to abort the incomplete write.
		_ = w.Close()
		return nil, fmt.Errorf("storage: GCS write failed: %w", err)
	}

	// Close flushes remaining data and finalises the object. Any server-side
	// error (permissions, quota, etc.) surfaces here.
	if err := w.Close(); err != nil {
		return nil, fmt.Errorf("storage: GCS close failed: %w", err)
	}

	url := fmt.Sprintf("https://storage.googleapis.com/%s/%s", s.bucket, key)
	return &UploadResult{URL: url, Key: key}, nil
}

// Delete removes the object at key. It is idempotent — a missing key is not
// treated as an error, matching the Storage contract.
func (s *gcsStorage) Delete(ctx context.Context, key string) error {
	obj := s.client.Bucket(s.bucket).Object(key)
	err := obj.Delete(ctx)
	if err != nil && err != storage.ErrObjectNotExist {
		return fmt.Errorf("storage: GCS delete failed: %w", err)
	}
	return nil
}
