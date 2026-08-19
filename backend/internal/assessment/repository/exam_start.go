package repository

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math/rand/v2"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"github.com/edugraph-ai/edugraph/internal/assessment/dto"
)

const pgUniqueViolation = "23505"

// AttemptSummary is the minimal shape StartAttempt/resume need -- lighter
// than the full StartAttemptResponse DTO, which also carries the
// question list.
type AttemptSummary struct {
	AttemptID        uuid.UUID
	AttemptNumber    int
	StartedAt        time.Time
	ExpiresAt        *time.Time
	TimeLimitMinutes *int
}

// FindInProgressAttempt returns the caller's currently-open attempt for
// this exam, if any. StartAttempt calling this first (and returning it
// unchanged rather than creating a new one) is what makes /start
// idempotent/resumable -- a refreshed browser calling /start again gets
// the exact same attempt, question order, and expiry back.
func (r *Repository) FindInProgressAttempt(ctx context.Context, studentID, examID uuid.UUID) (*AttemptSummary, error) {
	const q = `
		SELECT id, attempt_number, started_at, expires_at, time_limit_minutes
		FROM assessment.exam_attempts
		WHERE student_id = $1 AND exam_id = $2 AND status = 'in_progress'
		ORDER BY attempt_number DESC
		LIMIT 1
	`
	var a AttemptSummary
	err := r.pool.QueryRow(ctx, q, studentID, examID).Scan(&a.AttemptID, &a.AttemptNumber, &a.StartedAt, &a.ExpiresAt, &a.TimeLimitMinutes)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("find in-progress attempt: %w", err)
	}
	return &a, nil
}

// CountFinishedAttempts counts attempts that are no longer in_progress
// (submitted/auto_submitted/expired/cancelled/invalidated) -- what
// attempt_limit is actually checked against, so a currently-open attempt
// (already returned by FindInProgressAttempt, and about to be resumed
// rather than counted as "used up") doesn't double-count itself.
func (r *Repository) CountFinishedAttempts(ctx context.Context, studentID, examID uuid.UUID) (int, error) {
	const q = `
		SELECT count(*) FROM assessment.exam_attempts
		WHERE student_id = $1 AND exam_id = $2 AND status != 'in_progress'
	`
	var n int
	if err := r.pool.QueryRow(ctx, q, studentID, examID).Scan(&n); err != nil {
		return 0, fmt.Errorf("count finished attempts: %w", err)
	}
	return n, nil
}

type questionForRandomization struct {
	ID      uuid.UUID
	Letters []string
}

func (r *Repository) fetchQuestionIDsForRandomization(ctx context.Context, examID uuid.UUID) ([]questionForRandomization, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, options FROM assessment.questions WHERE exam_id = $1 ORDER BY sequence_number
	`, examID)
	if err != nil {
		return nil, fmt.Errorf("fetch questions for randomization: %w", err)
	}
	defer rows.Close()

	var out []questionForRandomization
	for rows.Next() {
		var id uuid.UUID
		var rawOptions []byte
		if err := rows.Scan(&id, &rawOptions); err != nil {
			return nil, fmt.Errorf("scan question for randomization: %w", err)
		}
		q := questionForRandomization{ID: id}
		if rawOptions != nil {
			var opts []dto.QuestionOption
			if err := json.Unmarshal(rawOptions, &opts); err != nil {
				return nil, fmt.Errorf("unmarshal options for question %s: %w", id, err)
			}
			for _, o := range opts {
				q.Letters = append(q.Letters, o.Letter)
			}
		}
		out = append(out, q)
	}
	return out, rows.Err()
}

// CreateAttemptWithForm generates and persists this student's exact exam
// form (question display order + per-question option display order) and
// creates the in_progress attempt, all in one transaction. If a
// concurrent request already created attempt_number in the same instant
// (two rapid clicks on "Start"), the unique index on
// (student_id, exam_id, attempt_number) rejects the second insert; that
// specific race is caught and resolved by returning the attempt the
// other request just created, via a recursive call to
// FindInProgressAttempt (not treated as an error).
func (r *Repository) CreateAttemptWithForm(
	ctx context.Context, studentID, examID, schoolID uuid.UUID,
	attemptNumber int, timeLimitMinutes *int, idempotencyKey *uuid.UUID,
) (*dto.StartAttemptResponse, error) {
	questions, err := r.fetchQuestionIDsForRandomization(ctx, examID)
	if err != nil {
		return nil, err
	}
	if len(questions) == 0 {
		return nil, fmt.Errorf("exam %s has no questions to start an attempt for", examID)
	}

	order := rand.Perm(len(questions))

	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin start attempt tx: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	now := time.Now().UTC()
	var expiresAt *time.Time
	if timeLimitMinutes != nil {
		e := now.Add(time.Duration(*timeLimitMinutes) * time.Minute)
		expiresAt = &e
	}

	var attemptID uuid.UUID
	insertErr := tx.QueryRow(ctx, `
		INSERT INTO assessment.exam_attempts
			(student_id, exam_id, school_id, attempt_number, started_at, expires_at,
			 time_limit_minutes, status, last_activity_at, idempotency_key)
		VALUES ($1, $2, $3, $4, $5, $6, $7, 'in_progress', $5, $8)
		RETURNING id
	`, studentID, examID, schoolID, attemptNumber, now, expiresAt, timeLimitMinutes, idempotencyKey).Scan(&attemptID)
	if insertErr != nil {
		var pgErr *pgconn.PgError
		if errors.As(insertErr, &pgErr) && pgErr.Code == pgUniqueViolation {
			// Lost the race to a concurrent start -- the winner's
			// attempt is what the caller should get back, not an error.
			return nil, errAttemptRaceLost
		}
		return nil, fmt.Errorf("create attempt: %w", insertErr)
	}

	for i, qID := range order {
		q := questions[qID]
		var optionOrderJSON []byte
		if len(q.Letters) > 0 {
			shuffled := append([]string(nil), q.Letters...)
			rand.Shuffle(len(shuffled), func(a, b int) { shuffled[a], shuffled[b] = shuffled[b], shuffled[a] })
			optionOrderJSON, err = json.Marshal(shuffled)
			if err != nil {
				return nil, fmt.Errorf("marshal option order: %w", err)
			}
		}
		if _, err := tx.Exec(ctx, `
			INSERT INTO assessment.attempt_questions (attempt_id, question_id, display_order, option_order)
			VALUES ($1, $2, $3, $4)
		`, attemptID, q.ID, i, optionOrderJSON); err != nil {
			return nil, fmt.Errorf("insert attempt question: %w", err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit start attempt tx: %w", err)
	}

	studentQuestions, err := r.FetchAttemptQuestions(ctx, attemptID)
	if err != nil {
		return nil, err
	}

	return &dto.StartAttemptResponse{
		AttemptID:        attemptID,
		ExamID:           examID,
		AttemptNumber:    attemptNumber,
		StartedAt:        now,
		ExpiresAt:        expiresAt,
		TimeLimitMinutes: timeLimitMinutes,
		Questions:        studentQuestions,
	}, nil
}

// errAttemptRaceLost is a sentinel the service layer checks for to
// re-fetch and return the winning concurrent attempt instead of
// surfacing an internal error to the student.
var errAttemptRaceLost = errors.New("lost race to create attempt")

// ErrAttemptRace exposes errAttemptRaceLost for the service layer.
func ErrAttemptRace() error { return errAttemptRaceLost }

// FetchAttemptQuestions returns this attempt's persisted question order
// with each question's options reordered per attempt_questions.option_order
// -- never re-randomized, so a browser refresh always sees the exact same
// form. The option LETTER (the stable answer identifier grading compares
// against) is unchanged; only which slot it renders in is reordered.
func (r *Repository) FetchAttemptQuestions(ctx context.Context, attemptID uuid.UUID) ([]dto.StudentQuestion, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT q.id, q.sequence_number, q.question_text, q.question_type, q.marks, q.part_label,
		       q.options, aq.display_order, aq.option_order
		FROM assessment.attempt_questions aq
		JOIN assessment.questions q ON q.id = aq.question_id
		WHERE aq.attempt_id = $1
		ORDER BY aq.display_order
	`, attemptID)
	if err != nil {
		return nil, fmt.Errorf("fetch attempt questions: %w", err)
	}
	defer rows.Close()

	var out []dto.StudentQuestion
	for rows.Next() {
		var sq dto.StudentQuestion
		var rawOptions, rawOptionOrder []byte
		var displayOrder int
		if err := rows.Scan(
			&sq.ID, &sq.SequenceNumber, &sq.QuestionText, &sq.QuestionType, &sq.Marks, &sq.PartLabel,
			&rawOptions, &displayOrder, &rawOptionOrder,
		); err != nil {
			return nil, fmt.Errorf("scan attempt question: %w", err)
		}
		// SequenceNumber becomes this attempt's display position, not the
		// original document order -- the frontend numbers questions 1..N
		// in the order the student actually sees them.
		sq.SequenceNumber = displayOrder + 1

		var opts []dto.QuestionOption
		if rawOptions != nil {
			if err := json.Unmarshal(rawOptions, &opts); err != nil {
				return nil, fmt.Errorf("unmarshal options for question %s: %w", sq.ID, err)
			}
		}
		sq.Options = reorderOptions(opts, rawOptionOrder)
		out = append(out, sq)
	}
	return out, rows.Err()
}

// reorderOptions applies the persisted per-attempt letter order. Falls
// back to the parsed (unshuffled) order if optionOrder is absent/invalid
// rather than failing the whole question -- a display-order glitch must
// never block a student from seeing/answering a question.
func reorderOptions(opts []dto.QuestionOption, rawOptionOrder []byte) []dto.QuestionOption {
	if len(rawOptionOrder) == 0 || len(opts) == 0 {
		return opts
	}
	var order []string
	if err := json.Unmarshal(rawOptionOrder, &order); err != nil {
		return opts
	}
	byLetter := make(map[string]dto.QuestionOption, len(opts))
	for _, o := range opts {
		byLetter[o.Letter] = o
	}
	reordered := make([]dto.QuestionOption, 0, len(opts))
	seen := make(map[string]bool, len(opts))
	for _, letter := range order {
		if o, ok := byLetter[letter]; ok {
			reordered = append(reordered, o)
			seen[letter] = true
		}
	}
	// Any option not covered by a stale/mismatched order array is
	// appended rather than dropped.
	for _, o := range opts {
		if !seen[o.Letter] {
			reordered = append(reordered, o)
		}
	}
	return reordered
}
