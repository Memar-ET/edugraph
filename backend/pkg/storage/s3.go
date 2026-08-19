package storage

import (
	"context"
	"fmt"
	"io"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/google/uuid"
)

// S3Storage implements StorageProvider using AWS S3.
// Upload stores the file under a freshly-generated UUID key and returns
// that key as the reference (matching the PostgresStorage contract, where
// the ref is a UUID string). Download fetches the object and streams it
// back to the caller -- the caller is responsible for closing the reader.
type S3Storage struct {
	client *s3.Client
	bucket string
}

// NewS3Storage loads AWS credentials from the environment / instance profile
// (standard SDK credential chain) and returns a ready-to-use S3Storage for
// the given region and bucket name.
func NewS3Storage(ctx context.Context, region, bucket string) (*S3Storage, error) {
	cfg, err := awsconfig.LoadDefaultConfig(ctx, awsconfig.WithRegion(region))
	if err != nil {
		return nil, fmt.Errorf("load aws config: %w", err)
	}
	return &S3Storage{
		client: s3.NewFromConfig(cfg),
		bucket: bucket,
	}, nil
}

func (s *S3Storage) Upload(ctx context.Context, fileName, mimeType string, file io.Reader) (string, error) {
	key := uuid.NewString() + "/" + fileName
	_, err := s.client.PutObject(ctx, &s3.PutObjectInput{
		Bucket:      aws.String(s.bucket),
		Key:         aws.String(key),
		Body:        file,
		ContentType: aws.String(mimeType),
	})
	if err != nil {
		return "", fmt.Errorf("s3 put object: %w", err)
	}
	return key, nil
}

func (s *S3Storage) Download(ctx context.Context, ref string) (io.ReadCloser, error) {
	out, err := s.client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(ref),
	})
	if err != nil {
		return nil, fmt.Errorf("s3 get object: %w", err)
	}
	return out.Body, nil
}
