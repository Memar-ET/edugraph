package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime"
	"path/filepath"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/redis/go-redis/v9" // Assuming go-redis is used

	"github.com/edugraph-ai/edugraph/internal/curriculum/dto"
	"github.com/edugraph-ai/edugraph/internal/curriculum/repository"
	apperrors "github.com/edugraph-ai/edugraph/pkg/errors"
	"github.com/edugraph-ai/edugraph/pkg/storage"
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
	job, err := s.repo.GetJob(ctx, jobID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, apperrors.NotFound("job not found")
		}
		return nil, err
	}
	return job, nil
}

// DownloadFile streams the original uploaded file for a job -- backs the
// Step 3 "view original PDF alongside the parsed tree" dev-mode proxy
// endpoint (GET /api/v1/storage/files/{jobId}). In prod mode this same
// StorageProvider would instead be one that returns a presigned S3 URL;
// swapping that in is the only change needed here.
func (s *Service) DownloadFile(ctx context.Context, jobID uuid.UUID) (io.ReadCloser, string, string, error) {
	fileRef, fileName, err := s.repo.GetFileRef(ctx, jobID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, "", "", apperrors.NotFound("job not found")
		}
		return nil, "", "", fmt.Errorf("fetch file reference: %w", err)
	}

	reader, err := s.storage.Download(ctx, fileRef)
	if err != nil {
		return nil, "", "", fmt.Errorf("download file: %w", err)
	}

	mimeType := mime.TypeByExtension(filepath.Ext(fileName))
	if mimeType == "" {
		mimeType = "application/octet-stream"
	}
	return reader, fileName, mimeType, nil
}

// Approve is Step 3's final act: the curriculum officer has reviewed (and
// possibly corrected) the AI-parsed tree and is ready to make it official.
//
// If `edited` is provided (the officer changed something in the UI), that
// becomes the structure that gets promoted AND the new value of
// upload_jobs.parsed_structure. If nil, the tree already stored from
// parsing is used unchanged.
func (s *Service) Approve(ctx context.Context, userID, jobID uuid.UUID, edited *dto.ParsedStructurePayload) (*dto.ApproveResponse, error) {
	status, storedStructure, err := s.repo.GetJobForApproval(ctx, jobID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, apperrors.NotFound("job not found")
		}
		return nil, fmt.Errorf("fetch job: %w", err)
	}
	if status != "parsed" && status != "review" {
		return nil, apperrors.Conflict(fmt.Sprintf(
			"job status is %q; only jobs with status 'parsed' or 'review' can be approved", status,
		))
	}

	var structure dto.ParsedStructurePayload
	var finalJSON []byte

	if edited != nil {
		structure = *edited
		finalJSON, err = json.Marshal(edited)
		if err != nil {
			return nil, apperrors.BadRequest("invalid parsedStructure in request body")
		}
	} else {
		if len(storedStructure) == 0 {
			return nil, apperrors.BadRequest(
				"job has no parsed structure yet; wait for parsing to finish, or submit parsedStructure in the request body",
			)
		}
		if err := json.Unmarshal(storedStructure, &structure); err != nil {
			return nil, fmt.Errorf("parse stored structure: %w", err)
		}
		finalJSON = storedStructure
	}

	if len(structure.Units) == 0 {
		return nil, apperrors.BadRequest("parsed structure has no units to promote")
	}

	return s.repo.ApproveAndPromote(ctx, jobID, userID, structure, finalJSON)
}
