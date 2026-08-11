package repository

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"github.com/edugraph-ai/edugraph/internal/curriculum/dto"
	"github.com/edugraph-ai/edugraph/pkg/pagination"
)

// ListJobsByUser backs the Curriculum Officer dashboard (feature 1.2):
// every upload job the given user has submitted, newest first, so they
// can track parsing/review/approval status without knowing a job's URL.
func (r *Repository) ListJobsByUser(ctx context.Context, userID uuid.UUID, p pagination.Params) ([]dto.JobListItem, int64, error) {
	var total int64
	if err := r.pool.QueryRow(ctx,
		`SELECT count(*) FROM curriculum.upload_jobs WHERE uploaded_by = $1`, userID,
	).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count jobs for user %s: %w", userID, err)
	}

	rows, err := r.pool.Query(ctx, `
		SELECT id, status, file_name, subject_code, grade_level, academic_year, created_at, approved_at
		FROM curriculum.upload_jobs
		WHERE uploaded_by = $1
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3
	`, userID, p.Limit, p.Offset())
	if err != nil {
		return nil, 0, fmt.Errorf("list jobs for user %s: %w", userID, err)
	}
	defer rows.Close()

	items := make([]dto.JobListItem, 0, p.Limit)
	for rows.Next() {
		var item dto.JobListItem
		if err := rows.Scan(
			&item.JobID, &item.Status, &item.FileName, &item.SubjectCode,
			&item.GradeLevel, &item.AcademicYear, &item.CreatedAt, &item.ApprovedAt,
		); err != nil {
			return nil, 0, fmt.Errorf("scan job row: %w", err)
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("iterate job rows: %w", err)
	}

	return items, total, nil
}
