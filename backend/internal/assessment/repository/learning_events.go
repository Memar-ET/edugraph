package repository

import (
	"context"
	"fmt"

	"github.com/google/uuid"
)

// RecordLearningEvents inserts (or, on a regrade, updates) one
// students.learning_events row per graded answer on an attempt -- the
// EG-GCKT Student Event Model (spec section 14). Runs at the same seam as
// RecomputeAttemptTotals, once an attempt is confirmed fully graded, so
// every event's correctness is always known at write time (see the
// column's nullability comment in V039 for why it's kept nullable anyway
// -- future non-exam event sources, not this path).
//
// skill_ids resolves via assessment.item_skill_mappings (the versioned
// Q-matrix, Milestone 0) when a mapping exists, falling back to the
// question's single questions.topic_id pointer otherwise -- exactly
// mirroring how ai-service's existing fetch_topic_mastery resolves a
// question to a topic today, so a question authored before EG-GCKT
// shipped (and never given an explicit Q-matrix row) still produces a
// usable skill_ids array via the same fallback the rest of the system
// already relies on.
//
// One event per assessment.student_answers row, keyed by answer_id --
// ON CONFLICT (answer_id) DO UPDATE makes a teacher regrade update the
// existing event's correctness/skill_ids/response rather than
// accumulating a duplicate "the student answered again" event (see
// V039's partial unique index for why that's correct, not a violation of
// the append-only-event principle).
func (r *Repository) RecordLearningEvents(ctx context.Context, attemptID uuid.UUID) error {
	const q = `
		INSERT INTO students.learning_events
			(student_id, school_id, item_id, answer_id, occurred_at, skill_ids, response, correctness,
			 resource_context, response_time_secs, assessment_id, item_version)
		SELECT
			sa.student_id, sa.school_id, sa.question_id, sa.id, sa.updated_at,
			COALESCE(
				(SELECT array_agg(DISTINCT m.topic_id) FROM assessment.item_skill_mappings m
				 WHERE m.question_id = sa.question_id AND m.is_current),
				CASE WHEN q.topic_id IS NOT NULL THEN ARRAY[q.topic_id] ELSE '{}'::uuid[] END
			),
			jsonb_build_object('answerText', sa.answer_text, 'answerData', sa.answer_data),
			sa.passed,
			'exam',
			sa.time_spent_secs,
			ea.exam_id,
			(SELECT max(m.version) FROM assessment.item_skill_mappings m
			 WHERE m.question_id = sa.question_id AND m.is_current)
		FROM assessment.student_answers sa
		JOIN assessment.questions q ON q.id = sa.question_id
		JOIN assessment.exam_attempts ea ON ea.id = sa.attempt_id
		WHERE sa.attempt_id = $1 AND sa.marks_awarded IS NOT NULL
		ON CONFLICT (answer_id) WHERE answer_id IS NOT NULL DO UPDATE SET
			skill_ids = EXCLUDED.skill_ids,
			response = EXCLUDED.response,
			correctness = EXCLUDED.correctness,
			response_time_secs = EXCLUDED.response_time_secs,
			item_version = EXCLUDED.item_version
	`
	if _, err := r.pool.Exec(ctx, q, attemptID); err != nil {
		return fmt.Errorf("record learning events for attempt %s: %w", attemptID, err)
	}
	return nil
}
