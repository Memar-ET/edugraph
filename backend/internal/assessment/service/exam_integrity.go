package service

import (
	"context"

	"github.com/google/uuid"

	"github.com/edugraph-ai/edugraph/internal/assessment/dto"
	apperrors "github.com/edugraph-ai/edugraph/pkg/errors"
)

// ReportIntegrityEvents records client-observed integrity signals (tab
// visibility, fullscreen, connection status) for the caller's own
// attempt. Best-effort/diagnostic by design -- see Part 20's explicit
// instruction that these are signals for troubleshooting/integrity
// review, never automatic proof of misconduct, so this never blocks or
// alters grading.
func (s *Service) ReportIntegrityEvents(ctx context.Context, userID, examID uuid.UUID, req dto.ReportIntegrityEventsRequest) error {
	student, err := s.repo.FetchStudentProfile(ctx, userID)
	if err != nil {
		return apperrors.Internal(err)
	}
	attempt, err := s.repo.FindLatestAttempt(ctx, student.ID, examID)
	if err != nil {
		return apperrors.Internal(err)
	}
	if attempt == nil {
		return apperrors.Conflict("no exam session found to attach these events to")
	}
	return s.repo.SaveIntegrityEvents(ctx, attempt.AttemptID, req.Events)
}

// IntegritySummaryEntry is the teacher-facing shape -- a plain count per
// event type, e.g. "12 tab-hidden events across 8 attempts," never a
// per-student accusation.
func (s *Service) GetExamIntegritySummary(ctx context.Context, userID, examID uuid.UUID) (map[string]int, error) {
	if err := s.verifyCallerOwnsExam(ctx, userID, examID); err != nil {
		return nil, err
	}
	rows, err := s.repo.FetchIntegritySummaryByExam(ctx, examID)
	if err != nil {
		return nil, apperrors.Internal(err)
	}
	out := make(map[string]int, len(rows))
	for _, r := range rows {
		out[r.EventType] = r.Count
	}
	return out, nil
}
