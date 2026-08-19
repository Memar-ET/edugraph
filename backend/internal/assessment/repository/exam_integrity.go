package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/edugraph-ai/edugraph/internal/assessment/dto"
)

// FindLatestAttempt returns the caller's most recent attempt for this
// exam regardless of status -- unlike FindInProgressAttempt, integrity
// events (a tab-hidden/connection-lost signal near submission) are
// diagnostic, best-effort, and may legitimately arrive just after
// submission finalizes the attempt, so this must not require
// in_progress specifically.
func (r *Repository) FindLatestAttempt(ctx context.Context, studentID, examID uuid.UUID) (*AttemptSummary, error) {
	const q = `
		SELECT id, attempt_number, started_at, expires_at, time_limit_minutes
		FROM assessment.exam_attempts
		WHERE student_id = $1 AND exam_id = $2
		ORDER BY attempt_number DESC
		LIMIT 1
	`
	var a AttemptSummary
	err := r.pool.QueryRow(ctx, q, studentID, examID).Scan(&a.AttemptID, &a.AttemptNumber, &a.StartedAt, &a.ExpiresAt, &a.TimeLimitMinutes)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("find latest attempt: %w", err)
	}
	return &a, nil
}

// SaveIntegrityEvents batch-inserts integrity signals for an attempt.
// ON CONFLICT DO NOTHING on (attempt_id, sequence_number) makes a
// resent/retried batch idempotent rather than duplicating events.
func (r *Repository) SaveIntegrityEvents(ctx context.Context, attemptID uuid.UUID, events []dto.IntegrityEventInput) error {
	if len(events) == 0 {
		return nil
	}
	batch := &pgx.Batch{}
	for _, e := range events {
		var metadata []byte
		if len(e.Metadata) > 0 {
			metadata = e.Metadata
		}
		batch.Queue(`
			INSERT INTO assessment.exam_integrity_events
				(attempt_id, event_type, occurred_at, sequence_number, metadata)
			VALUES ($1, $2, $3, $4, $5)
			ON CONFLICT (attempt_id, sequence_number) DO NOTHING
		`, attemptID, e.EventType, e.OccurredAt, e.SequenceNumber, metadata)
	}
	br := r.pool.SendBatch(ctx, batch)
	defer br.Close()
	for range events {
		if _, err := br.Exec(); err != nil {
			return fmt.Errorf("save integrity event: %w", err)
		}
	}
	return nil
}

// IntegritySummary is a per-attempt count of each event type -- what the
// teacher-facing quality/analytics view surfaces as neutral signals, per
// the explicit "N visibility changes, not accusations of cheating"
// framing.
type IntegritySummary struct {
	EventType string
	Count     int
}

// FetchIntegritySummaryByExam aggregates integrity events across every
// attempt on an exam, grouped by event type -- backs the teacher-facing
// exam-level summary (ExamQualityPage).
func (r *Repository) FetchIntegritySummaryByExam(ctx context.Context, examID uuid.UUID) ([]IntegritySummary, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT ie.event_type, count(*)
		FROM assessment.exam_integrity_events ie
		JOIN assessment.exam_attempts ea ON ea.id = ie.attempt_id
		WHERE ea.exam_id = $1
		GROUP BY ie.event_type
	`, examID)
	if err != nil {
		return nil, fmt.Errorf("fetch integrity summary: %w", err)
	}
	defer rows.Close()

	out := make([]IntegritySummary, 0)
	for rows.Next() {
		var s IntegritySummary
		if err := rows.Scan(&s.EventType, &s.Count); err != nil {
			return nil, fmt.Errorf("scan integrity summary row: %w", err)
		}
		out = append(out, s)
	}
	return out, rows.Err()
}
