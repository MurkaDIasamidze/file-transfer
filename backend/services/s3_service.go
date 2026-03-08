// services/s3_service.go
package services

import (
	"bytes"
	"context"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

// S3Service handles all file operations with AWS S3.
type S3Service struct {
	client *s3.Client
	bucket string
}

// NewS3Service creates a new S3Service.
// Call this once at startup and reuse the instance everywhere.
func NewS3Service(region, accessKey, secretKey, bucket string) (*S3Service, error) {
	// Create AWS config with explicit credentials from .env
	cfg, err := config.LoadDefaultConfig(
		context.TODO(),
		config.WithRegion(region),
		config.WithCredentialsProvider(
			credentials.NewStaticCredentialsProvider(accessKey, secretKey, ""),
		),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to load AWS config: %w", err)
	}

	return &S3Service{
		client: s3.NewFromConfig(cfg),
		bucket: bucket,
	}, nil
}

// Upload sends a file to S3.
//
//   key      — the path inside the bucket, e.g. "users/42/report.pdf"
//   data     — raw file bytes
//   mimeType — content type, e.g. "application/pdf"
//
// Returns the S3 key (same as input) so you can store it in the database.
func (s *S3Service) Upload(ctx context.Context, key string, data []byte, mimeType string) (string, error) {
	_, err := s.client.PutObject(ctx, &s3.PutObjectInput{
		Bucket:      aws.String(s.bucket),
		Key:         aws.String(key),
		Body:        bytes.NewReader(data),
		ContentType: aws.String(mimeType),
	})
	if err != nil {
		return "", fmt.Errorf("s3 upload failed for key %q: %w", key, err)
	}
	return key, nil
}

// Delete removes a file from S3 permanently.
func (s *S3Service) Delete(ctx context.Context, key string) error {
	_, err := s.client.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return fmt.Errorf("s3 delete failed for key %q: %w", key, err)
	}
	return nil
}

// PresignDownload generates a temporary download URL for a file.
// The URL is valid for `ttl` duration (e.g. 15 minutes).
// Send this URL to the frontend — the browser downloads directly from S3,
// bypassing your server entirely.
func (s *S3Service) PresignDownload(ctx context.Context, key string, ttl time.Duration) (string, error) {
	presigner := s3.NewPresignClient(s.client)

	req, err := presigner.PresignGetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(key),
	}, s3.WithPresignExpires(ttl))
	if err != nil {
		return "", fmt.Errorf("failed to presign download for key %q: %w", key, err)
	}

	return req.URL, nil
}

// BuildKey creates a consistent S3 key for a user's file.
//
// Examples:
//   BuildKey(42, 0,  "report.pdf")         → "users/42/report.pdf"
//   BuildKey(42, 7,  "docs/notes.txt")     → "users/42/folders/7/docs/notes.txt"
func BuildKey(userID uint, folderID uint, relPath string) string {
	if folderID == 0 {
		return fmt.Sprintf("users/%d/%s", userID, relPath)
	}
	return fmt.Sprintf("users/%d/folders/%d/%s", userID, folderID, relPath)
}