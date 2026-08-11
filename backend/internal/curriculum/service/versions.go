package service

import (
	"context"

	"github.com/edugraph-ai/edugraph/internal/curriculum/dto"
)

// MarkAsRevision links an already-approved subject code as the version
// that supersedes another -- see Repository.MarkAsRevision.
func (s *Service) MarkAsRevision(ctx context.Context, newCode, previousCode string) (*dto.SubjectVersion, error) {
	return s.repo.MarkAsRevision(ctx, newCode, previousCode)
}

// GetVersionHistory returns a subject's revision lineage, oldest first.
func (s *Service) GetVersionHistory(ctx context.Context, code string) ([]dto.SubjectVersion, error) {
	return s.repo.GetVersionHistory(ctx, code)
}
