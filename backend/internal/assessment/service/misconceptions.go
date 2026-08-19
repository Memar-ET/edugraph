package service

import (
	"context"
	"errors"

	"github.com/google/uuid"

	"github.com/edugraph-ai/edugraph/internal/assessment/dto"
	"github.com/edugraph-ai/edugraph/internal/assessment/repository"
	apperrors "github.com/edugraph-ai/edugraph/pkg/errors"
)

// ListCandidateMisconceptions backs the teacher-facing review queue
// (EG-GCKT Milestone 6) -- every unreviewed hypothesis at the caller's
// school, across all students/topics.
func (s *Service) ListCandidateMisconceptions(ctx context.Context, userID uuid.UUID) ([]dto.MisconceptionHypothesis, error) {
	schoolID, err := s.repo.TeacherSchoolID(ctx, userID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, apperrors.NotFound("user not found")
		}
		return nil, apperrors.Internal(err)
	}

	hypotheses, err := s.repo.ListCandidateMisconceptions(ctx, schoolID)
	if err != nil {
		return nil, apperrors.Internal(err)
	}
	return hypotheses, nil
}

// ReviewMisconception confirms or rejects a candidate hypothesis.
// Confirming folds it into the student's skill_states.misconception_state
// -- the only path anything ever moves from "LLM guessed" to "a teacher
// actually believes this is real."
func (s *Service) ReviewMisconception(ctx context.Context, userID, misconceptionID uuid.UUID, req dto.ReviewMisconceptionRequest) (*dto.MisconceptionHypothesis, error) {
	schoolID, err := s.repo.TeacherSchoolID(ctx, userID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, apperrors.NotFound("user not found")
		}
		return nil, apperrors.Internal(err)
	}

	hypothesis, err := s.repo.ReviewMisconception(ctx, schoolID, misconceptionID, userID, req.Decision)
	if errors.Is(err, repository.ErrMisconceptionNotFound) {
		return nil, apperrors.NotFound("misconception hypothesis not found, or already reviewed")
	}
	if err != nil {
		return nil, apperrors.Internal(err)
	}
	return hypothesis, nil
}
