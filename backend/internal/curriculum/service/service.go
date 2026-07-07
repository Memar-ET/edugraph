package service

import (
	"context"
	"fmt"
	"io"

	"github.com/edugraph-ai/edugraph/internal/curriculum/dto"
	"github.com/edugraph-ai/edugraph/internal/curriculum/repository"
	"github.com/edugraph-ai/edugraph/pkg/storage"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9" // Assuming go-redis is used
)

type Service struct {
	repo    *repository.Repository
	storage storage.StorageProvider
	redis   *redis.Client
}

func New(repo *repository.Repository, storage storage.StorageProvider, redis *redis.Client) *Service {
	return &Service{repo: repo, storage: storage, redis: redis}
}

// Upload handles the entire ingestion workflow:
// 1. Save file to storage (Postgres/S3)
// 2. Create DB record
// 3. Notify AI Service via Redis
func (s *Service) Upload(ctx context.Context, userID uuid.UUID, req dto.UploadRequest, fileName, mimeType string, fileSize int64, file io.Reader) (*dto.UploadResponse, error) {
	// 1. Save to Storage (The "Fork in the Road")
	// If Dev: Saves to Postgres BYTEA. Returns UUID.
	// If Prod: Saves to S3. Returns S3 Key.
	fileRef, err := s.storage.Upload(ctx, fileName, mimeType, file)
	if err != nil {
		return nil, fmt.Errorf("storage upload failed: %w", err)
	}

	// 2. Create Job Record in Postgres
	jobID, err := s.repo.CreateJob(ctx, userID, req, fileRef, fileName, fileSize)
	if err != nil {
		return nil, fmt.Errorf("create job record failed: %w", err)
	}

	// 3. Push to Redis Queue for AI Service
	// The AI service (Python) listens on "queue:curriculum:parse"
	err = s.redis.LPush(ctx, "queue:curriculum:parse", jobID.String()).Err()
	if err != nil {
		// Non-fatal: The job exists, but the queue failed.
		// A background cron can pick up "pending" jobs later.
		fmt.Printf("⚠️ Redis queue push failed for job %s: %v\n", jobID, err)
	}

	return &dto.UploadResponse{
		JobID:   jobID,
		Status:  "pending",
		Message: "File uploaded successfully. Parsing queued.",
	}, nil
}

func (s *Service) GetJob(ctx context.Context, jobID uuid.UUID) (*dto.JobStatus, error) {
	return s.repo.GetJob(ctx, jobID)
}
