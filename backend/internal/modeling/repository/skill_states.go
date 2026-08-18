package repository

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"github.com/edugraph-ai/edugraph/internal/modeling/dto"
)

// FetchAllSkillStates returns every students.skill_states row for the
// given student, newest first -- backs GET /students/me/skill-states
// (the student-facing skill-map view). Only topics with at least one
// piece of evidence appear (cold start via row absence, same as
// FetchSkillState above) -- an honestly-empty list for a brand new
// student, not a fabricated full-curriculum row set.
func (r *Repository) FetchAllSkillStates(ctx context.Context, studentID uuid.UUID) ([]dto.SkillState, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT ss.topic_id, t.title_en, t.subject_code, t.grade_level,
		       ss.mastery_probability, ss.mastery_status, ss.uncertainty,
		       ss.trend, ss.evidence_count, ss.forgetting_risk, ss.computed_at
		FROM students.skill_states ss
		JOIN curriculum.topics t ON t.id = ss.topic_id
		WHERE ss.student_id = $1
		ORDER BY ss.computed_at DESC
	`, studentID)
	if err != nil {
		return nil, fmt.Errorf("fetch all skill states: %w", err)
	}
	defer rows.Close()

	out := make([]dto.SkillState, 0)
	for rows.Next() {
		var s dto.SkillState
		if err := rows.Scan(
			&s.TopicID, &s.TopicTitle, &s.SubjectCode, &s.GradeLevel,
			&s.MasteryProbability, &s.MasteryStatus, &s.Uncertainty,
			&s.Trend, &s.EvidenceCount, &s.ForgettingRisk, &s.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan skill state row: %w", err)
		}
		out = append(out, s)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate skill state rows: %w", err)
	}
	return out, nil
}
