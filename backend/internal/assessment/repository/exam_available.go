package repository

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"github.com/edugraph-ai/edugraph/internal/assessment/dto"
)

// FetchAvailableExams returns every published exam matching the given
// school/grade -- the same scoping verifyStudentAccess enforces per-exam
// (service.verifyStudentAccess) -- newest first, with whether the
// student has already submitted each one. published_at was added by
// V051 without a backfill, so pre-existing published exams have it NULL;
// created_at is used as the honest fallback rather than scanning a NULL
// into a non-pointer time.Time.
func (r *Repository) FetchAvailableExams(ctx context.Context, schoolID uuid.UUID, gradeLevel int, studentID uuid.UUID) ([]dto.ExamAvailability, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT
			e.id, e.title, e.subject_code, e.grade_level, e.exam_scope, e.total_marks,
			(SELECT count(*) FROM assessment.questions q WHERE q.exam_id = e.id),
			COALESCE(e.published_at, e.created_at), e.closed_at,
			EXISTS (
				SELECT 1 FROM assessment.exam_attempts a
				WHERE a.exam_id = e.id AND a.student_id = $3 AND a.submitted_at IS NOT NULL
			)
		FROM assessment.exams e
		WHERE e.school_id = $1 AND e.grade_level = $2 AND e.status = 'published'
		ORDER BY COALESCE(e.published_at, e.created_at) DESC
	`, schoolID, gradeLevel, studentID)
	if err != nil {
		return nil, fmt.Errorf("fetch available exams: %w", err)
	}
	defer rows.Close()

	items := make([]dto.ExamAvailability, 0)
	for rows.Next() {
		var item dto.ExamAvailability
		if err := rows.Scan(
			&item.ExamID, &item.Title, &item.SubjectCode, &item.GradeLevel, &item.ExamScope, &item.TotalMarks,
			&item.QuestionCount, &item.PublishedAt, &item.ClosesAt, &item.AlreadyAttempted,
		); err != nil {
			return nil, fmt.Errorf("scan available exam row: %w", err)
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate available exam rows: %w", err)
	}
	return items, nil
}
