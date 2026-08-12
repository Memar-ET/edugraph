package service

import (
	"context"

	"github.com/edugraph-ai/edugraph/internal/curriculum/dto"
)

// ListSubjects returns every promoted subject system-wide -- backs the
// Ministry curriculum browser.
func (s *Service) ListSubjects(ctx context.Context) ([]dto.SubjectListItem, error) {
	return s.repo.ListSubjects(ctx)
}
