package service

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"github.com/edugraph-ai/edugraph/internal/assessment/dto"
	apperrors "github.com/edugraph-ai/edugraph/pkg/errors"
)


// CloseExam moves a published exam to lifecycle_status='closed' so no further
// student submissions are accepted. Only the exam owner (teacher/school_admin)
// may close it; the exam must currently be in 'published' state.
func (s *Service) CloseExam(ctx context.Context, callerID, examID uuid.UUID) (*dto.CloseExamResponse, error) {
	if err := s.verifyCallerOwnsExam(ctx, callerID, examID); err != nil {
		return nil, err
	}
	if err := s.repo.CloseExam(ctx, examID); err != nil {
		return nil, apperrors.Internal(fmt.Errorf("close exam: %w", err))
	}
	return &dto.CloseExamResponse{
		ExamID:  examID,
		Status:  "closed",
		Message: "Exam closed. No further submissions will be accepted.",
	}, nil
}

// AutosaveExamDraft upserts a student's in-progress answers into
// exam_draft_answers so they can recover from a browser disconnect without
// losing their work. Reuses the shared verifyStudentAccess check (exam must
// be published, student's school/grade must match).
func (s *Service) AutosaveExamDraft(ctx context.Context, studentID, examID uuid.UUID, req dto.AutosaveRequest) error {
	if _, _, err := s.verifyStudentAccess(ctx, studentID, examID); err != nil {
		return err
	}
	return s.repo.UpsertDraftAnswers(ctx, studentID, examID, req.Answers)
}

// GetExamDraft returns the student's most recently autosaved answers for an
// exam. Drafts are scoped to student_id so no cross-student leakage is
// possible; we still verify the exam exists before returning.
func (s *Service) GetExamDraft(ctx context.Context, studentID, examID uuid.UUID) ([]dto.DraftAnswerItem, error) {
	return s.repo.FetchDraftAnswers(ctx, studentID, examID)
}
