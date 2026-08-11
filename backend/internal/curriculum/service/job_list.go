package service

import (
	"context"

	"github.com/google/uuid"

	"github.com/edugraph-ai/edugraph/internal/curriculum/dto"
	"github.com/edugraph-ai/edugraph/pkg/pagination"
)

// ListJobs returns userID's own upload jobs, newest first -- backs the
// Curriculum Officer dashboard (feature 1.2).
func (s *Service) ListJobs(ctx context.Context, userID uuid.UUID, p pagination.Params) ([]dto.JobListItem, int64, error) {
	return s.repo.ListJobsByUser(ctx, userID, p)
}
