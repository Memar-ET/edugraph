package service

import (
	"context"
	"errors"

	"github.com/google/uuid"

	"github.com/edugraph-ai/edugraph/internal/assessment/dto"
	"github.com/edugraph-ai/edugraph/internal/assessment/repository"
	apperrors "github.com/edugraph-ai/edugraph/pkg/errors"
)

// StartAttempt is the exam-taking entry point: creates (or, if the
// student already has one open, resumes) the student's exam session.
// Calling it twice while an attempt is in_progress is idempotent -- it
// returns the exact same attempt/question order/expiry, never a new one,
// which is what makes a browser refresh mid-exam safe. This is also the
// only place question/option order is ever generated -- everything else
// (ListExamQuestionsForStudent, SubmitExam) reads the persisted form.
func (s *Service) StartAttempt(ctx context.Context, userID, examID uuid.UUID) (*dto.StartAttemptResponse, error) {
	exam, student, err := s.verifyStudentAccess(ctx, userID, examID)
	if err != nil {
		return nil, err
	}

	if existing, err := s.repo.FindInProgressAttempt(ctx, student.ID, examID); err != nil {
		return nil, apperrors.Internal(err)
	} else if existing != nil {
		return s.attemptResponseFor(ctx, examID, existing)
	}

	finished, err := s.repo.CountFinishedAttempts(ctx, student.ID, examID)
	if err != nil {
		return nil, apperrors.Internal(err)
	}
	if finished >= exam.AttemptLimit {
		return nil, apperrors.Forbidden("you have used all allowed attempts for this exam")
	}

	resp, err := s.repo.CreateAttemptWithForm(ctx, student.ID, examID, student.SchoolID, finished+1, exam.TimeLimitMinutes, nil)
	if errors.Is(err, repository.ErrAttemptRace()) {
		// Lost a start/start race to a concurrent request -- the winner's
		// attempt is now in_progress; return that instead of erroring.
		existing, err := s.repo.FindInProgressAttempt(ctx, student.ID, examID)
		if err != nil {
			return nil, apperrors.Internal(err)
		}
		if existing == nil {
			return nil, apperrors.Internal(errors.New("attempt race reported but no in-progress attempt found"))
		}
		return s.attemptResponseFor(ctx, examID, existing)
	}
	if err != nil {
		return nil, apperrors.Internal(err)
	}
	return resp, nil
}

func (s *Service) attemptResponseFor(ctx context.Context, examID uuid.UUID, a *repository.AttemptSummary) (*dto.StartAttemptResponse, error) {
	questions, err := s.repo.FetchAttemptQuestions(ctx, a.AttemptID)
	if err != nil {
		return nil, apperrors.Internal(err)
	}
	return &dto.StartAttemptResponse{
		AttemptID:        a.AttemptID,
		ExamID:           examID,
		AttemptNumber:    a.AttemptNumber,
		StartedAt:        a.StartedAt,
		ExpiresAt:        a.ExpiresAt,
		TimeLimitMinutes: a.TimeLimitMinutes,
		Questions:        questions,
	}, nil
}
