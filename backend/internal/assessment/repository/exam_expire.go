package repository

import (
	"context"
	"fmt"

	"github.com/google/uuid"
)

// ExpiredAttempt is one row FetchExpiredInProgressAttempts finds --
// enough for the auto-submit worker to grade/finalize it without a
// student-facing userID (there is no caller here, the worker acts on
// the attempt's own student_id/school_id directly).
type ExpiredAttempt struct {
	AttemptID uuid.UUID
	ExamID    uuid.UUID
	StudentID uuid.UUID
	SchoolID  uuid.UUID
}

// FetchExpiredInProgressAttempts finds every attempt whose server-set
// expires_at has passed but which the student never explicitly
// submitted -- what the auto-submit ticker (internal/assessment/
// examworker) sweeps on a short interval. Attempts with no
// time_limit_minutes (expires_at IS NULL) never appear here -- an
// untimed exam has no expiry to enforce.
func (r *Repository) FetchExpiredInProgressAttempts(ctx context.Context) ([]ExpiredAttempt, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, exam_id, student_id, school_id
		FROM assessment.exam_attempts
		WHERE status = 'in_progress' AND expires_at IS NOT NULL AND expires_at < now()
	`)
	if err != nil {
		return nil, fmt.Errorf("fetch expired in-progress attempts: %w", err)
	}
	defer rows.Close()

	var out []ExpiredAttempt
	for rows.Next() {
		var a ExpiredAttempt
		if err := rows.Scan(&a.AttemptID, &a.ExamID, &a.StudentID, &a.SchoolID); err != nil {
			return nil, fmt.Errorf("scan expired attempt: %w", err)
		}
		out = append(out, a)
	}
	return out, rows.Err()
}
