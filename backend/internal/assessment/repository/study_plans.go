package repository

import (
	"context"
	"fmt"

	"github.com/edugraph-ai/edugraph/internal/assessment/dto"
	"github.com/google/uuid"
)

// FetchActiveStudyPlans returns the student's active plans, newest first
// (one per target: the generator deactivates the previous plan for the
// same target exam when it writes a new one).
func (r *Repository) FetchActiveStudyPlans(ctx context.Context, studentID uuid.UUID) ([]dto.StudyPlan, error) {
	const q = `
		SELECT id, target_exam_id, plan_data, total_days, total_hours,
		       language, generated_at, expires_at
		FROM students.study_plans
		WHERE student_id = $1 AND is_active
		ORDER BY generated_at DESC
	`
	rows, err := r.pool.Query(ctx, q, studentID)
	if err != nil {
		return nil, fmt.Errorf("fetch study plans: %w", err)
	}
	defer rows.Close()

	out := make([]dto.StudyPlan, 0)
	for rows.Next() {
		var p dto.StudyPlan
		var planData []byte
		if err := rows.Scan(
			&p.ID, &p.TargetExamID, &planData, &p.TotalDays, &p.TotalHours,
			&p.Language, &p.GeneratedAt, &p.ExpiresAt,
		); err != nil {
			return nil, fmt.Errorf("scan study plan: %w", err)
		}
		p.PlanData = planData
		out = append(out, p)
	}
	return out, rows.Err()
}
