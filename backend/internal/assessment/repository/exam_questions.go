package repository

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"

	"github.com/edugraph-ai/edugraph/internal/assessment/dto"
)

// FetchQuestionsForStudent is the student-facing counterpart of
// FetchQuestionsForGrading -- same table, but the SELECT list deliberately
// excludes answer_key/clo_code/clo_align_* so a student taking the exam
// never receives the correct answer over the wire.
func (r *Repository) FetchQuestionsForStudent(ctx context.Context, examID uuid.UUID) ([]dto.StudentQuestion, error) {
	const q = `
		SELECT id, sequence_number, question_text, question_type, marks, part_label, options
		FROM assessment.questions
		WHERE exam_id = $1
		ORDER BY sequence_number
	`
	rows, err := r.pool.Query(ctx, q, examID)
	if err != nil {
		return nil, fmt.Errorf("fetch questions for student: %w", err)
	}
	defer rows.Close()

	var out []dto.StudentQuestion
	for rows.Next() {
		var sq dto.StudentQuestion
		var rawOptions []byte
		if err := rows.Scan(&sq.ID, &sq.SequenceNumber, &sq.QuestionText, &sq.QuestionType, &sq.Marks, &sq.PartLabel, &rawOptions); err != nil {
			return nil, fmt.Errorf("scan question for student: %w", err)
		}
		if rawOptions != nil {
			if err := json.Unmarshal(rawOptions, &sq.Options); err != nil {
				return nil, fmt.Errorf("unmarshal options for question %s: %w", sq.ID, err)
			}
		}
		out = append(out, sq)
	}
	return out, rows.Err()
}
