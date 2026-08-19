package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/edugraph-ai/edugraph/internal/assessment/dto"
)

var ErrMisconceptionNotFound = errors.New("misconception hypothesis not found")

// ListCandidateMisconceptions returns every 'candidate' hypothesis for a
// school, newest first -- the teacher review queue (EG-GCKT Milestone 6).
func (r *Repository) ListCandidateMisconceptions(ctx context.Context, schoolID uuid.UUID) ([]dto.MisconceptionHypothesis, error) {
	const q = `
		SELECT m.id, m.student_id, m.topic_id, t.title_en, m.misconception_text, m.trigger_pattern,
		       m.confidence, m.status, m.intervention_text, m.generated_by_model, m.created_at
		FROM students.misconception_hypotheses m
		JOIN curriculum.topics t ON t.id = m.topic_id
		WHERE m.school_id = $1 AND m.status = 'candidate'
		ORDER BY m.created_at DESC
	`
	rows, err := r.pool.Query(ctx, q, schoolID)
	if err != nil {
		return nil, fmt.Errorf("list candidate misconceptions: %w", err)
	}
	defer rows.Close()

	out := make([]dto.MisconceptionHypothesis, 0)
	for rows.Next() {
		var m dto.MisconceptionHypothesis
		if err := rows.Scan(
			&m.ID, &m.StudentID, &m.TopicID, &m.TopicTitle, &m.MisconceptionText, &m.TriggerPattern,
			&m.Confidence, &m.Status, &m.InterventionText, &m.GeneratedByModel, &m.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan misconception hypothesis: %w", err)
		}
		out = append(out, m)
	}
	return out, rows.Err()
}

// ReviewMisconception flips a candidate hypothesis to confirmed/rejected.
// Only a 'candidate' row can be reviewed -- confirming/rejecting an
// already-decided one is rejected as ErrMisconceptionNotFound (matched
// zero rows) rather than silently allowed to flip back and forth.
func (r *Repository) ReviewMisconception(ctx context.Context, schoolID, misconceptionID, reviewerID uuid.UUID, decision string) (*dto.MisconceptionHypothesis, error) {
	const q = `
		UPDATE students.misconception_hypotheses m
		SET status = $3, reviewed_by = $4, reviewed_at = now()
		FROM curriculum.topics t
		WHERE m.id = $1 AND m.school_id = $2 AND m.status = 'candidate' AND t.id = m.topic_id
		RETURNING m.id, m.student_id, m.topic_id, t.title_en, m.misconception_text, m.trigger_pattern,
		          m.confidence, m.status, m.intervention_text, m.generated_by_model, m.created_at
	`
	var out dto.MisconceptionHypothesis
	err := r.pool.QueryRow(ctx, q, misconceptionID, schoolID, decision, reviewerID).Scan(
		&out.ID, &out.StudentID, &out.TopicID, &out.TopicTitle, &out.MisconceptionText, &out.TriggerPattern,
		&out.Confidence, &out.Status, &out.InterventionText, &out.GeneratedByModel, &out.CreatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrMisconceptionNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("review misconception: %w", err)
	}

	if decision == "confirmed" {
		if _, err := r.pool.Exec(ctx, `
			UPDATE students.skill_states
			SET misconception_state = COALESCE(misconception_state, '[]'::jsonb) || jsonb_build_array(
				jsonb_build_object('id', $3::text, 'text', $4::text, 'confirmedAt', now())
			)
			WHERE student_id = $1 AND topic_id = $2
		`, out.StudentID, out.TopicID, out.ID.String(), out.MisconceptionText); err != nil {
			return nil, fmt.Errorf("fold confirmed misconception into skill_states: %w", err)
		}
	}

	return &out, nil
}
