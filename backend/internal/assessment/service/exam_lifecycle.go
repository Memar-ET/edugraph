package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/edugraph-ai/edugraph/internal/assessment/dto"
	"github.com/edugraph-ai/edugraph/internal/assessment/repository"
	apperrors "github.com/edugraph-ai/edugraph/pkg/errors"
)

// CloseExam moves a published exam to status='closed' so no further
// student submissions are accepted. Only the exam owner (teacher/
// school_admin) may close it; the exam must currently be in 'published'
// state.
func (s *Service) CloseExam(ctx context.Context, callerID, examID uuid.UUID) (*dto.CloseExamResponse, error) {
	if err := s.verifyCallerOwnsExam(ctx, callerID, examID); err != nil {
		return nil, err
	}
	if err := s.repo.CloseExam(ctx, examID); err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			// Repository.CloseExam's WHERE status='published' matched
			// zero rows -- either the exam doesn't exist, or (the common
			// case) it's already closed/never published. Previously this
			// fell through to the unconditional apperrors.Internal below,
			// surfacing a raw 500 for an entirely expected "already
			// closed" double-click instead of a clear 409.
			return nil, apperrors.Conflict("exam not found or is not currently published")
		}
		return nil, apperrors.Internal(fmt.Errorf("close exam: %w", err))
	}
	s.repo.RecordAuditAction(ctx, "exam.close", "exam", examID.String())
	return &dto.CloseExamResponse{
		ExamID:  examID,
		Status:  "closed",
		Message: "Exam closed. No further submissions will be accepted.",
	}, nil
}

// AutosaveExamDraft upserts a student's in-progress answers into
// exam_draft_answers so they can recover from a browser disconnect without
// losing their work. Requires an in_progress, unexpired attempt (created
// by StartAttempt) -- autosave after submission or past expires_at is
// rejected, not silently accepted.
//
// Previously passed the JWT userID straight through as if it were
// students.id (discarding verifyStudentAccess's own resolved student.ID)
// -- exam_draft_answers.student_id is a real FK to students(id), a
// different table/ID space, so every autosave call would have failed
// its foreign-key constraint against Supabase's real schema. Fixed by
// using the resolved student.ID.
func (s *Service) AutosaveExamDraft(ctx context.Context, userID, examID uuid.UUID, req dto.AutosaveRequest) error {
	_, student, err := s.verifyStudentAccess(ctx, userID, examID)
	if err != nil {
		return err
	}
	attempt, err := s.repo.FindInProgressAttempt(ctx, student.ID, examID)
	if err != nil {
		return apperrors.Internal(err)
	}
	if attempt == nil {
		return apperrors.Conflict("no exam session in progress -- start the exam first")
	}
	if attempt.ExpiresAt != nil && !time.Now().UTC().Before(*attempt.ExpiresAt) {
		return apperrors.Conflict("time has expired for this attempt")
	}
	return s.repo.UpsertDraftAnswers(ctx, attempt.AttemptID, student.ID, examID, req.Answers)
}

// GetExamDraft returns the student's most recently autosaved answers for
// their current in_progress attempt. Same userID-resolution fix as
// AutosaveExamDraft above (student.ID, not the raw JWT userID). Returns
// an empty draft (not an error) when there's no in_progress attempt --
// e.g. the exam-instructions page probing before the student has
// clicked Start.
func (s *Service) GetExamDraft(ctx context.Context, userID, examID uuid.UUID) (*dto.ExamDraftResponse, error) {
	student, err := s.repo.FetchStudentProfile(ctx, userID)
	if err != nil {
		return nil, apperrors.Internal(err)
	}
	attempt, err := s.repo.FindInProgressAttempt(ctx, student.ID, examID)
	if err != nil {
		return nil, apperrors.Internal(err)
	}
	if attempt == nil {
		return &dto.ExamDraftResponse{ExamID: examID, Answers: []dto.DraftAnswerItem{}}, nil
	}

	answers, err := s.repo.FetchDraftAnswers(ctx, attempt.AttemptID)
	if err != nil {
		return nil, err
	}
	savedAt := time.Time{}
	for _, a := range answers {
		if a.SavedAt.After(savedAt) {
			savedAt = a.SavedAt
		}
	}
	return &dto.ExamDraftResponse{ExamID: examID, Answers: answers, SavedAt: savedAt}, nil
}
