package repository

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/edugraph-ai/edugraph/internal/assessment/dto"
)

// CloseExam sets lifecycle_status='closed' and closed_at=now() on an exam
// that is currently 'published'. Returns ErrNotFound if the exam doesn't
// exist in a closable state.
func (r *Repository) CloseExam(ctx context.Context, examID uuid.UUID) error {
	tag, err := r.pool.Exec(ctx, `
		UPDATE assessment.exams
		SET lifecycle_status = 'closed', closed_at = now(), updated_at = now()
		WHERE id = $1 AND lifecycle_status IN ('published', 'approved')
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
// assessment.exam_draft_answers, keyed on (student_id, exam_id, question_id).
// Existing rows for the same question are overwritten with the latest answer.
func (r *Repository) UpsertDraftAnswers(ctx context.Context, studentID, examID uuid.UUID, answers []dto.AutosaveDraftAnswer) error {
	if len(answers) == 0 {
		return nil
	}
	batch := &pgx.Batch{}
	for _, a := range answers {
		batch.Queue(`
			INSERT INTO assessment.exam_draft_answers
				(student_id, exam_id, question_id, selected_option_id, text_answer, saved_at)
			VALUES ($1, $2, $3, $4, $5, now())
			ON CONFLICT (student_id, exam_id, question_id) DO UPDATE SET
				selected_option_id = EXCLUDED.selected_option_id,
				text_answer        = EXCLUDED.text_answer,
				saved_at           = now()
		`, studentID, examID, a.QuestionID, a.SelectedOptionID, a.TextAnswer)
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

// FetchDraftAnswers returns all saved draft answers for a (student, exam) pair,
// ordered by the time they were last saved.
func (r *Repository) FetchDraftAnswers(ctx context.Context, studentID, examID uuid.UUID) ([]dto.DraftAnswerItem, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT question_id, selected_option_id, text_answer
		FROM assessment.exam_draft_answers
		WHERE student_id = $1 AND exam_id = $2
		ORDER BY saved_at
	`, studentID, examID)
	if err != nil {
		return nil, fmt.Errorf("fetch draft answers: %w", err)
	}
	defer rows.Close()

	var out []dto.DraftAnswerItem
	for rows.Next() {
		var item dto.DraftAnswerItem
		if err := rows.Scan(&item.QuestionID, &item.SelectedOptionID, &item.TextAnswer); err != nil {
			return nil, fmt.Errorf("scan draft answer: %w", err)
		}
		out = append(out, item)
	}
	return out, rows.Err()
}
