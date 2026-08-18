package repository

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"github.com/edugraph-ai/edugraph/internal/assessment/dto"
	"github.com/edugraph-ai/edugraph/pkg/pagination"
)

// ListExamsBySchool backs GET /api/v1/exams -- every exam belonging to the
// caller's school, newest first. School-scoped rather than
// created_by-scoped: ClassAnalyticsPage/QuestionBankPage need to see exams
// a colleague at the same school uploaded (e.g. building a shared question
// bank), not just the caller's own uploads, matching how TeacherSchoolID/
// ExamSchoolID already scope exam ownership checks elsewhere (checklist
// 11.3) at the school level, not the individual-teacher level.
func (r *Repository) ListExamsBySchool(ctx context.Context, schoolID uuid.UUID, p pagination.Params) ([]dto.ExamListItem, int64, error) {
	var total int64
	if err := r.pool.QueryRow(ctx,
		`SELECT count(*) FROM assessment.exams WHERE school_id = $1`, schoolID,
	).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count exams for school %s: %w", schoolID, err)
	}

	rows, err := r.pool.Query(ctx, `
		SELECT
			e.id, e.title, e.subject_code, e.grade_level, e.status, e.exam_scope,
			e.total_marks, e.created_at,
			(SELECT count(*) FROM assessment.questions q WHERE q.exam_id = e.id),
			(SELECT count(*) FROM assessment.exam_attempts a WHERE a.exam_id = e.id AND a.submitted_at IS NOT NULL)
		FROM assessment.exams e
		WHERE e.school_id = $1
		ORDER BY e.created_at DESC
		LIMIT $2 OFFSET $3
	`, schoolID, p.Limit, p.Offset())
	if err != nil {
		return nil, 0, fmt.Errorf("list exams for school %s: %w", schoolID, err)
	}
	defer rows.Close()

	items := make([]dto.ExamListItem, 0, p.Limit)
	for rows.Next() {
		var item dto.ExamListItem
		if err := rows.Scan(
			&item.ExamID, &item.Title, &item.SubjectCode, &item.GradeLevel, &item.Status, &item.ExamScope,
			&item.TotalMarks, &item.CreatedAt, &item.QuestionCount, &item.SubmissionCount,
		); err != nil {
			return nil, 0, fmt.Errorf("scan exam row: %w", err)
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("iterate exam rows: %w", err)
	}

	return items, total, nil
}
