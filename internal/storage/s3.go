package storage

import (
	"context"
	"fmt"
	"io"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

// S3Config holds the credentials and location for an S3 bucket.
type S3Config struct {
	AccessKeyID     string
	SecretAccessKey string
	Region          string
	Bucket          string
}

// s3Storage implements Storage backed by AWS S3.
type s3Storage struct {
	client *s3.Client
	bucket string
	region string
}

// NewS3Storage constructs an S3-backed Storage.
// It uses static credentials from cfg — appropriate for server-side use where
// the IAM key is loaded from environment variables / secrets manager.
func NewS3Storage(cfg S3Config) (Storage, error) {
	awsCfg, err := awsconfig.LoadDefaultConfig(
		context.Background(),
		awsconfig.WithRegion(cfg.Region),
		awsconfig.WithCredentialsProvider(
			credentials.NewStaticCredentialsProvider(
				cfg.AccessKeyID,
				cfg.SecretAccessKey,
				"", // session token — not needed for long-lived IAM keys
			),
		),
	)
	if err != nil {
		return nil, fmt.Errorf("storage: failed to load AWS config: %w", err)
	}

	client := s3.NewFromConfig(awsCfg)
	return &s3Storage{client: client, bucket: cfg.Bucket, region: cfg.Region}, nil
}

// Upload streams r to S3 at the given key with the given content type.
// The object is created with public-read ACL so the URL works in a browser
// without any pre-signing step.
func (s *s3Storage) Upload(ctx context.Context, key string, r io.Reader, contentType string) (*UploadResult, error) {
	_, err := s.client.PutObject(ctx, &s3.PutObjectInput{
		Bucket:      aws.String(s.bucket),
		Key:         aws.String(key),
		Body:        r,
		ContentType: aws.String(contentType),
	})
	if err != nil {
		return nil, fmt.Errorf("storage: S3 PutObject failed: %w", err)
	}

	url := fmt.Sprintf("https://%s.s3.%s.amazonaws.com/%s", s.bucket, s.region, key)
	return &UploadResult{URL: url, Key: key}, nil
}

// Delete removes the object at key. It is idempotent — a missing key is not
// treated as an error, matching the Storage contract.
func (s *s3Storage) Delete(ctx context.Context, key string) error {
	_, err := s.client.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return fmt.Errorf("storage: S3 DeleteObject failed: %w", err)
	}
	return nil
}
