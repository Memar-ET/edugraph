package service

import (
	"context"
	"errors"

	"github.com/google/uuid"

	"github.com/edugraph-ai/edugraph/internal/assessment/dto"
	"github.com/edugraph-ai/edugraph/internal/assessment/repository"
	apperrors "github.com/edugraph-ai/edugraph/pkg/errors"
)

// GetAvailableExams returns every published exam available to the calling
// student -- GET /api/v1/students/me/available-exams, the exam-taking
// entry point (StudentExamListPage etc.).
func (s *Service) GetAvailableExams(ctx context.Context, userID uuid.UUID) (*dto.AvailableExamsResponse, error) {
	student, err := s.repo.FetchStudentProfile(ctx, userID)
	if errors.Is(err, repository.ErrNotFound) {
		return nil, apperrors.Forbidden("no student profile for this account")
	}
	if err != nil {
		return nil, apperrors.Internal(err)
	}

	exams, err := s.repo.FetchAvailableExams(ctx, student.SchoolID, student.GradeLevel, student.ID)
	if err != nil {
		return nil, apperrors.Internal(err)
	}

	return &dto.AvailableExamsResponse{Exams: exams}, nil
}
