package storage

import (
	"bytes"
	"context"
	"fmt"
	"io"

	"github.com/jackc/pgx/v5/pgxpool"
)

type PostgresStorage struct {
	pool *pgxpool.Pool
}

func NewPostgresStorage(pool *pgxpool.Pool) *PostgresStorage {
	return &PostgresStorage{pool: pool}
}

func (s *PostgresStorage) Upload(ctx context.Context, fileName, mimeType string, file io.Reader) (string, error) {
	// Read file into memory (acceptable for Dev/Local, NOT for massive Prod files)
	// For very large files, we would stream this, but BYTEA requires the byte slice.
	buf := new(bytes.Buffer)
	size, err := buf.ReadFrom(file)
	if err != nil {
		return "", fmt.Errorf("read file data: %w", err)
	}

	var id string
	query := `
		INSERT INTO app_storage.local_files (file_name, mime_type, file_data, size_bytes)
		VALUES ($1, $2, $3, $4)
		RETURNING id
	`
	err = s.pool.QueryRow(ctx, query, fileName, mimeType, buf.Bytes(), size).Scan(&id)
	if err != nil {
		return "", fmt.Errorf("insert local file: %w", err)
	}

	return id, nil // Return the UUID as the reference
}

func (s *PostgresStorage) Download(ctx context.Context, ref string) (io.ReadCloser, error) {
	var data []byte
	query := `SELECT file_data FROM app_storage.local_files WHERE id = $1`
	err := s.pool.QueryRow(ctx, query, ref).Scan(&data)
	if err != nil {
		return nil, fmt.Errorf("fetch local file: %w", err)
	}
	return io.NopCloser(bytes.NewReader(data)), nil
}
