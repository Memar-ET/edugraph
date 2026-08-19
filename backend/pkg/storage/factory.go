package storage

import (
	"context"
	"fmt"
	"os"

	"github.com/jackc/pgx/v5/pgxpool"
)

// NewFromEnv constructs the appropriate StorageProvider based on the
// STORAGE_PROVIDER environment variable ("postgres" or "s3"). When
// STORAGE_PROVIDER is unset or "postgres", PostgresStorage is returned --
// this is the dev default and never requires AWS credentials.
// When STORAGE_PROVIDER is "s3", S3Storage is returned using AWS_REGION and
// AWS_S3_CURRICULUM_BUCKET from the environment (standard SDK credential chain).
func NewFromEnv(ctx context.Context, pool *pgxpool.Pool) (StorageProvider, error) {
	provider := os.Getenv("STORAGE_PROVIDER")
	if provider == "" {
		provider = "postgres"
	}
	switch provider {
	case "postgres":
		return NewPostgresStorage(pool), nil
	case "s3":
		region := os.Getenv("AWS_REGION")
		if region == "" {
			region = "af-south-1"
		}
		bucket := os.Getenv("AWS_S3_CURRICULUM_BUCKET")
		if bucket == "" {
			return nil, fmt.Errorf("AWS_S3_CURRICULUM_BUCKET must be set when STORAGE_PROVIDER=s3")
		}
		return NewS3Storage(ctx, region, bucket)
	default:
		return nil, fmt.Errorf("unknown STORAGE_PROVIDER %q (valid: postgres, s3)", provider)
	}
}
