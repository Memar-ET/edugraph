package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/edugraph-ai/edugraph/internal/assessment/dto"
	"github.com/edugraph-ai/edugraph/internal/assessment/repository"
	apperrors "github.com/edugraph-ai/edugraph/pkg/errors"
	"github.com/google/uuid"
)

// GenerateStudyPlan queues Capability 3B generation for the authenticated
// student. Unlike the fire-and-forget gap-analysis push, the queue write
// IS the whole operation here, so a Redis failure is a real error.
func (s *Service) GenerateStudyPlan(ctx context.Context, userID uuid.UUID, req dto.GenerateStudyPlanRequest) (*dto.GenerateStudyPlanResponse, error) {
	student, err := s.repo.FetchStudentProfile(ctx, userID)
	if errors.Is(err, repository.ErrNotFound) {
		return nil, apperrors.Forbidden("no student profile for this account")
	}
	if err != nil {
		return nil, apperrors.Internal(err)
	}

	language := "en"
	if req.Language != nil {
		language = *req.Language
	}
	payload, err := json.Marshal(map[string]any{
		"studentId":    student.ID.String(),
		"schoolId":     student.SchoolID.String(),
		"targetExamId": req.TargetExamID,
		"language":     language,
	})
	if err != nil {
		return nil, apperrors.Internal(err)
	}

	if err := s.redis.LPush(ctx, "queue:studyplan:generate", payload).Err(); err != nil {
		return nil, apperrors.Internal(fmt.Errorf("queue study plan generation: %w", err))
	}

	return &dto.GenerateStudyPlanResponse{
		Status:  "queued",
		Message: "Study plan generation queued. Fetch GET /students/me/study-plans shortly.",
	}, nil
}

// ListMyStudyPlans returns the authenticated student's active plans.
func (s *Service) ListMyStudyPlans(ctx context.Context, userID uuid.UUID) ([]dto.StudyPlan, error) {
	student, err := s.repo.FetchStudentProfile(ctx, userID)
	if errors.Is(err, repository.ErrNotFound) {
		return nil, apperrors.Forbidden("no student profile for this account")
	}
	if err != nil {
		return nil, apperrors.Internal(err)
	}

	plans, err := s.repo.FetchActiveStudyPlans(ctx, student.ID)
	if err != nil {
		return nil, apperrors.Internal(err)
	}
	return plans, nil
}
