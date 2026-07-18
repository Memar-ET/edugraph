package service

import (
	"context"
	"errors"

	"github.com/edugraph-ai/edugraph/internal/assessment/dto"
	"github.com/edugraph-ai/edugraph/internal/assessment/repository"
	apperrors "github.com/edugraph-ai/edugraph/pkg/errors"
	"github.com/google/uuid"
)

// GetMyExamInsight returns the Capability 3A insight (narrative summary +
// granular gap records) for the authenticated student's attempt at an
// exam. The analysis is asynchronous -- a just-graded attempt may not
// have an insight yet, surfaced as a 404 the frontend can poll on.
func (s *Service) GetMyExamInsight(ctx context.Context, userID, examID uuid.UUID) (*dto.ExamInsight, error) {
	student, err := s.repo.FetchStudentProfile(ctx, userID)
	if errors.Is(err, repository.ErrNotFound) {
		return nil, apperrors.Forbidden("no student profile for this account")
	}
	if err != nil {
		return nil, apperrors.Internal(err)
	}

	insight, err := s.repo.FetchInsightForStudentExam(ctx, student.ID, examID)
	if errors.Is(err, repository.ErrNotFound) {
		return nil, apperrors.NotFound("no insight for this exam yet -- it is generated shortly after grading completes")
	}
	if err != nil {
		return nil, apperrors.Internal(err)
	}
	return insight, nil
}

// ListExamInsights backs the teacher-facing insight list for one exam --
// same role-gating-only convention as ListQuestionsForGrading (router
// RequireRole, no per-teacher-school ownership check).
func (s *Service) ListExamInsights(ctx context.Context, examID uuid.UUID) ([]dto.ExamInsightListEntry, error) {
	insights, err := s.repo.FetchInsightsForExam(ctx, examID)
	if err != nil {
		return nil, apperrors.Internal(err)
	}
	return insights, nil
}

// GetMySubjectProfiles returns the authenticated student's Subject Health
// Layer -- one rolling mastery score per subject across all analyzed exams.
func (s *Service) GetMySubjectProfiles(ctx context.Context, userID uuid.UUID) ([]dto.SubjectProfile, error) {
	student, err := s.repo.FetchStudentProfile(ctx, userID)
	if errors.Is(err, repository.ErrNotFound) {
		return nil, apperrors.Forbidden("no student profile for this account")
	}
	if err != nil {
		return nil, apperrors.Internal(err)
	}

	profiles, err := s.repo.FetchSubjectProfiles(ctx, student.ID)
	if err != nil {
		return nil, apperrors.Internal(err)
	}
	return profiles, nil
}
