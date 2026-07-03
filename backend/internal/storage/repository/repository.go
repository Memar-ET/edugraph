package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

const presignExpiry = 15 * time.Minute

type Repository struct {
	presign *s3.PresignClient
}

// New loads AWS credentials from the standard chain (env, shared config,
// IAM role). Credential resolution is lazy, so this succeeds even in local
// dev without real AWS credentials configured — presigning will only fail
// when actually invoked.
func New(ctx context.Context, region string) (*Repository, error) {
	cfg, err := config.LoadDefaultConfig(ctx, config.WithRegion(region))
	if err != nil {
		return nil, fmt.Errorf("load aws config: %w", err)
	}
	client := s3.NewFromConfig(cfg)
	return &Repository{presign: s3.NewPresignClient(client)}, nil
}

func (r *Repository) PresignUpload(ctx context.Context, bucket, key, contentType string) (string, time.Duration, error) {
	input := &s3.PutObjectInput{Bucket: aws.String(bucket), Key: aws.String(key)}
	if contentType != "" {
		input.ContentType = aws.String(contentType)
	}
	req, err := r.presign.PresignPutObject(ctx, input, s3.WithPresignExpires(presignExpiry))
	if err != nil {
		return "", 0, fmt.Errorf("presign upload: %w", err)
	}
	return req.URL, presignExpiry, nil
}

func (r *Repository) PresignDownload(ctx context.Context, bucket, key string) (string, time.Duration, error) {
	req, err := r.presign.PresignGetObject(ctx, &s3.GetObjectInput{Bucket: aws.String(bucket), Key: aws.String(key)}, s3.WithPresignExpires(presignExpiry))
	if err != nil {
		return "", 0, fmt.Errorf("presign download: %w", err)
	}
	return req.URL, presignExpiry, nil
}
