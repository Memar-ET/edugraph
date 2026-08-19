package repository

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/edugraph-ai/edugraph/internal/assessment/dto"
)

// CloseExam sets status='closed' on an exam that is currently 'published'.
// Returns ErrNotFound if the exam doesn't exist in a closable state.
//
// Previously guarded on lifecycle_status IN ('published','approved') and
// only wrote lifecycle_status/closed_at -- V051 added lifecycle_status
// with a DEFAULT 'draft' but nothing besides its own one-time backfill
// ever advances it (PublishExam only ever touches the status column), so
// that guard matched zero rows for every exam published through the
// normal flow and this endpoint 404'd unconditionally. Worse, even a
// "successful" close left status='published', which is the column
// verifyStudentAccess/autosave/draft actually check -- so closing an
// exam never stopped submissions either way. Fixed to guard on and set
// status, the one column every access-control check in this domain
// actually reads; lifecycle_status is left alone (dead, not migrated
// away in this pass) and closed_at still records when it happened.
func (r *Repository) CloseExam(ctx context.Context, examID uuid.UUID) error {
	tag, err := r.pool.Exec(ctx, `
		UPDATE assessment.exams
		SET status = 'closed', closed_at = now(), updated_at = now()
		WHERE id = $1 AND status = 'published'
	`, examID)
	if err != nil {
		return fmt.Errorf("close exam: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// UpsertDraftAnswers batch-upserts one row per answer into
// assessment.exam_draft_answers, keyed on (student_id, exam_id, question_id)
// -- unchanged from V052, but studentID here must be students.id (the
// caller previously passed the JWT userID straight through, which
// violated exam_draft_answers' FK to students(id) on every real call;
// see Service.AutosaveExamDraft). attemptID (V057) is stamped on each row
// so a future multi-attempt exam's drafts can be told apart per attempt;
// existing rows for the same question are overwritten with the latest
// answer either way.
func (r *Repository) UpsertDraftAnswers(ctx context.Context, attemptID, studentID, examID uuid.UUID, answers []dto.AutosaveDraftAnswer) error {
	if len(answers) == 0 {
		return nil
	}
	batch := &pgx.Batch{}
	for _, a := range answers {
		batch.Queue(`
			INSERT INTO assessment.exam_draft_answers
				(student_id, exam_id, question_id, attempt_id, response, saved_at)
			VALUES ($1, $2, $3, $4, $5, now())
			ON CONFLICT (student_id, exam_id, question_id) DO UPDATE SET
				attempt_id = EXCLUDED.attempt_id,
				response = EXCLUDED.response,
				saved_at = now()
		`, studentID, examID, a.QuestionID, attemptID, a.Response)
	}
	br := r.pool.SendBatch(ctx, batch)
	defer br.Close()
	for _, a := range answers {
		if _, err := br.Exec(); err != nil {
			return fmt.Errorf("upsert draft answer for question %s: %w", a.QuestionID, err)
		}
	}
	return nil
}

// FetchDraftAnswers returns all saved draft answers for the given
// attempt, ordered by the time they were last saved.
func (r *Repository) FetchDraftAnswers(ctx context.Context, attemptID uuid.UUID) ([]dto.DraftAnswerItem, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT question_id, response, saved_at
		FROM assessment.exam_draft_answers
		WHERE attempt_id = $1
		ORDER BY saved_at
	`, attemptID)
	if err != nil {
		return nil, fmt.Errorf("fetch draft answers: %w", err)
	}
	defer rows.Close()

	out := make([]dto.DraftAnswerItem, 0)
	for rows.Next() {
		var item dto.DraftAnswerItem
		if err := rows.Scan(&item.QuestionID, &item.Response, &item.SavedAt); err != nil {
			return nil, fmt.Errorf("scan draft answer: %w", err)
		}
		out = append(out, item)
	}
	return out, rows.Err()
}
