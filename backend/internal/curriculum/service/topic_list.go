package service

import (
	"context"

	"github.com/edugraph-ai/edugraph/internal/curriculum/dto"
)

// ListTopicsBySubject returns every topic under a subject -- backs the
// prerequisites UI's topic picker.
func (s *Service) ListTopicsBySubject(ctx context.Context, subjectCode string) ([]dto.TopicListItem, error) {
	return s.repo.ListTopicsBySubject(ctx, subjectCode)
}
