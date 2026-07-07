package storage

import (
	"context"
	"io"
)

// StorageProvider abstracts file storage.
// In Dev: Uses Postgres (BYTEA).
// In Prod: Uses AWS S3.
type StorageProvider interface {
	// Upload saves the file and returns a reference ID (UUID for Postgres, S3 Key for AWS).
	Upload(ctx context.Context, fileName string, mimeType string, file io.Reader) (string, error)

	// Download retrieves the file using the reference ID.
	Download(ctx context.Context, ref string) (io.ReadCloser, error)
}
