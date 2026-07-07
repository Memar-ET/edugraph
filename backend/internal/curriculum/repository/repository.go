package repository

import (
	"context"
	"fmt"

	"github.com/edugraph-ai/edugraph/internal/curriculum/dto"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository struct {
	pool *pgxpool.Pool
}

func New(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

// CreateJob inserts a new upload job into the database.
func (r *Repository) CreateJob(ctx context.Context, uploadedBy uuid.UUID, req dto.UploadRequest, fileRef string, fileName string, fileSize int64) (uuid.UUID, error) {
	var jobID uuid.UUID
	query := `
		INSERT INTO curriculum.upload_jobs 
			(uploaded_by, subject_code, grade_level, academic_year, file_s3_key, file_name, file_size_bytes, status)
		VALUES 
			($1, $2, $3, $4, $5, $6, $7, 'pending')
		RETURNING id
	`
	// Note: We use file_s3_key to store the Postgres UUID in Dev mode.
	// In Prod, this will be the S3 Key.
	err := r.pool.QueryRow(ctx, query,
		uploadedBy, req.SubjectCode, req.GradeLevel, req.AcademicYear,
		fileRef, fileName, fileSize,
	).Scan(&jobID)

	if err != nil {
		return uuid.Nil, fmt.Errorf("create upload job: %w", err)
	}
	return jobID, nil
}

// GetJob retrieves the status of a specific job.
func (r *Repository) GetJob(ctx context.Context, jobID uuid.UUID) (*dto.JobStatus, error) {
	var job dto.JobStatus
	var errStr *string
	query := `
		SELECT id, status, file_name, parse_error 
		FROM curriculum.upload_jobs 
		WHERE id = $1
	`
	err := r.pool.QueryRow(ctx, query, jobID).Scan(&job.JobID, &job.Status, &job.FileName, &errStr)
	if err != nil {
		return nil, err
	}
	job.Error = errStr
	return &job, nil
}
