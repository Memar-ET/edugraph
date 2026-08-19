# ADR-001: Dual-Storage Adapter Pattern (PostgresStorage dev / S3 future)

Date: 2026-07-17  
Status: Accepted

## Context

The curriculum pipeline needs to store uploaded PDF/DOCX files and serve them back to the frontend reviewer. The production design calls for S3-compatible object storage, but there is no AWS account or cloud infrastructure during local development. We needed a way to get the full pipeline working locally without AWS, while ensuring the eventual switch to S3 doesn't require changes throughout the handler/service layer.

## Decision

Introduce a `StorageProvider` interface in `backend/pkg/storage/interface.go`:

```go
type StorageProvider interface {
    Upload(ctx context.Context, fileName, mimeType string, file io.Reader) (ref string, error)
    Download(ctx context.Context, ref string) (io.ReadCloser, error)
}
```

The dev implementation (`PostgresStorage`, `backend/pkg/storage/postgres.go`) stores file bytes in `app_storage.local_files` as a BYTEA column and returns the row UUID as the storage reference. This reference is stored in `curriculum.upload_jobs.file_s3_key` (legacy column name from the S3-first design, kept as-is to avoid a migration).

No `S3StorageProvider` exists yet. Adding S3 support is a single new file that implements the interface; handlers and services need zero changes.

## Consequences

**Good:**
- The full curriculum upload/review/download pipeline works in local dev with no external dependencies.
- Swapping to S3 in production is a one-file addition (`S3StorageProvider`) and a config toggle.
- Handler and service code is decoupled from storage implementation details.

**Bad:**
- `file_s3_key` column name is misleading — it holds a Postgres UUID, not an S3 key.
- Storing binary files in Postgres BYTEA is not suitable for large-scale production use. Performance and cost will degrade as the file corpus grows.
- The `app_storage.local_files` table will need to be migrated or emptied before switching to S3 in production.
