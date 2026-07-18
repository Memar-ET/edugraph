package repository

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/edugraph-ai/edugraph/internal/assessment/dto"
)

type StudentProfile struct {
	ID         uuid.UUID
	SchoolID   uuid.UUID
	GradeLevel int
}

// FetchStudentProfile resolves students.id (what exam_attempts/
// student_answers actually FK to) from the authenticated user's id --
// users.id and students.id are different UUIDs, joined 1:1 via
// students.user_id.
func (r *Repository) FetchStudentProfile(ctx context.Context, userID uuid.UUID) (*StudentProfile, error) {
	const q = `SELECT id, school_id, grade_level FROM students WHERE user_id = $1`
	var p StudentProfile
	err := r.pool.QueryRow(ctx, q, userID).Scan(&p.ID, &p.SchoolID, &p.GradeLevel)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("fetch student profile: %w", err)
	}
	return &p, nil
}

// FindAttempt returns the existing attempt id for (student, exam), or nil
// if none exists yet. No unique DB constraint backs this -- "one attempt
// per student per exam" is a service-layer policy, checked here first.
func (r *Repository) FindAttempt(ctx context.Context, studentID, examID uuid.UUID) (*uuid.UUID, error) {
	const q = `SELECT id FROM assessment.exam_attempts WHERE student_id = $1 AND exam_id = $2 LIMIT 1`
	var id uuid.UUID
	err := r.pool.QueryRow(ctx, q, studentID, examID).Scan(&id)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("find attempt: %w", err)
	}
	return &id, nil
}

// CreateAttempt inserts a new exam_attempts row. Call FindAttempt first --
// this doesn't check for an existing one itself (see FindAttempt's
// comment on why there's no DB constraint).
func (r *Repository) CreateAttempt(ctx context.Context, studentID, examID, schoolID uuid.UUID, isOffline bool) (uuid.UUID, error) {
	const q = `
		INSERT INTO assessment.exam_attempts (student_id, exam_id, school_id, started_at, is_offline)
		VALUES ($1, $2, $3, now(), $4)
		RETURNING id
	`
	var id uuid.UUID
	if err := r.pool.QueryRow(ctx, q, studentID, examID, schoolID, isOffline).Scan(&id); err != nil {
		return uuid.Nil, fmt.Errorf("create attempt: %w", err)
	}
	return id, nil
}

type QuestionForGrading struct {
	ID             uuid.UUID
	SequenceNumber int
	QuestionText   string
	PartLabel      *string
	QuestionType   string
	Marks          int
	AnswerKey      map[string]string // {"correctOption": "B"}, nil if unset
	Options        []dto.QuestionOption
}

// FetchQuestionsForGrading is exam-scoped (not attempt-scoped) -- both
// flows need the full question list (text, type, possible marks, answer
// key) to grade against, regardless of which student/attempt is being
// graded. Also backs the teacher-facing grading UI's column headers
// (handler.ListExamQuestionsForGrading) -- unlike the student-facing
// ListExamQuestionsForStudent, this includes answer_key since it's useful
// grading reference for the teacher, not a leak.
func (r *Repository) FetchQuestionsForGrading(ctx context.Context, examID uuid.UUID) ([]QuestionForGrading, error) {
	const q = `
		SELECT id, sequence_number, question_text, part_label, question_type, marks, answer_key, options
		FROM assessment.questions WHERE exam_id = $1 ORDER BY sequence_number
	`
	rows, err := r.pool.Query(ctx, q, examID)
	if err != nil {
		return nil, fmt.Errorf("fetch questions for grading: %w", err)
	}
	defer rows.Close()

	var out []QuestionForGrading
	for rows.Next() {
		var qg QuestionForGrading
		var rawAnswerKey, rawOptions []byte
		if err := rows.Scan(&qg.ID, &qg.SequenceNumber, &qg.QuestionText, &qg.PartLabel, &qg.QuestionType, &qg.Marks, &rawAnswerKey, &rawOptions); err != nil {
			return nil, fmt.Errorf("scan question for grading: %w", err)
		}
		if rawOptions != nil {
			if err := json.Unmarshal(rawOptions, &qg.Options); err != nil {
				return nil, fmt.Errorf("unmarshal options for question %s: %w", qg.ID, err)
			}
		}
		if rawAnswerKey != nil {
			if err := json.Unmarshal(rawAnswerKey, &qg.AnswerKey); err != nil {
				return nil, fmt.Errorf("unmarshal answer key for question %s: %w", qg.ID, err)
			}
		}
		out = append(out, qg)
	}
	return out, rows.Err()
}

type GradedAnswer struct {
	QuestionID    uuid.UUID
	AnswerText    string
	MarksAwarded  *float64 // nil = needs_grading
	MarksPossible int
	Passed        *bool
	TimeSpentSecs *int // client-reported, nil when the client didn't time it
}

// SaveStudentAnswers upserts one attempt's answers in a single
// transaction, mirroring ApproveAndPromote's tx.Begin/loop/tx.Commit style
// (the established bulk-write pattern in this codebase -- no pgx.Batch/
// CopyFrom used anywhere else either). ON CONFLICT makes "Save All"
// idempotent (V020's unique constraint).
func (r *Repository) SaveStudentAnswers(ctx context.Context, attemptID, studentID, schoolID uuid.UUID, answers []GradedAnswer) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin save answers tx: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	const q = `
		INSERT INTO assessment.student_answers
			(attempt_id, student_id, question_id, school_id, answer_text, marks_awarded, marks_possible, passed, time_spent_secs)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		ON CONFLICT (attempt_id, question_id) DO UPDATE SET
			answer_text = EXCLUDED.answer_text,
			marks_awarded = EXCLUDED.marks_awarded,
			marks_possible = EXCLUDED.marks_possible,
			passed = EXCLUDED.passed,
			-- a later write without timing (e.g. a teacher bulk-grade of
			-- the same answer) must not erase the student's reported time
			time_spent_secs = COALESCE(EXCLUDED.time_spent_secs, assessment.student_answers.time_spent_secs),
			updated_at = now()
	`
	for _, a := range answers {
		if _, err := tx.Exec(ctx, q, attemptID, studentID, a.QuestionID, schoolID, a.AnswerText, a.MarksAwarded, a.MarksPossible, a.Passed, a.TimeSpentSecs); err != nil {
			return fmt.Errorf("save answer for question %s: %w", a.QuestionID, err)
		}
	}

	return tx.Commit(ctx)
}

// RecomputeAttemptTotals sums marks_awarded across every answer on an
// attempt. If any answer is still pending (marks_awarded IS NULL --
// "needs_grading" is inferred, there's no status column), the attempt
// stays unfinalized (total_score/percentage/passed left NULL). This is
// called after every answer write in BOTH flows, so Flow 2's bulk-grade
// endpoint finishing a pending Flow-1 essay answer finalizes the attempt
// too, without a third "finalize grading" endpoint.
func (r *Repository) RecomputeAttemptTotals(ctx context.Context, attemptID uuid.UUID) error {
	const passThreshold = 50.0

	const selectQ = `
		SELECT count(*) FILTER (WHERE marks_awarded IS NULL) AS pending,
		       COALESCE(sum(marks_awarded), 0) AS total_awarded,
		       COALESCE(sum(marks_possible), 0) AS total_possible
		FROM assessment.student_answers
		WHERE attempt_id = $1
	`
	var pending int
	var totalAwarded, totalPossible float64
	if err := r.pool.QueryRow(ctx, selectQ, attemptID).Scan(&pending, &totalAwarded, &totalPossible); err != nil {
		return fmt.Errorf("recompute attempt totals: %w", err)
	}

	if pending > 0 {
		// Still has ungraded answers -- leave totals unfinalized.
		return nil
	}

	percentage := 0.0
	if totalPossible > 0 {
		percentage = (totalAwarded / totalPossible) * 100
	}
	passed := percentage >= passThreshold

	const updateQ = `
		UPDATE assessment.exam_attempts
		SET total_score = $2, percentage = $3, passed = $4, submitted_at = COALESCE(submitted_at, now())
		WHERE id = $1
	`
	if _, err := r.pool.Exec(ctx, updateQ, attemptID, totalAwarded, percentage, passed); err != nil {
		return fmt.Errorf("finalize attempt totals: %w", err)
	}
	return nil
}

// FetchAttemptTotals reads back the finalized totals RecomputeAttemptTotals
// wrote, so a caller that just fully graded an attempt can return them
// without duplicating the sum logic.
func (r *Repository) FetchAttemptTotals(ctx context.Context, attemptID uuid.UUID) (totalScore, percentage *float64, passed *bool, err error) {
	const q = `SELECT total_score, percentage, passed FROM assessment.exam_attempts WHERE id = $1`
	if err := r.pool.QueryRow(ctx, q, attemptID).Scan(&totalScore, &percentage, &passed); err != nil {
		return nil, nil, nil, fmt.Errorf("fetch attempt totals: %w", err)
	}
	return totalScore, percentage, passed, nil
}
