package service

import (
	"context"

	"github.com/edugraph-ai/edugraph/internal/curriculum/dto"
)

// GetSubjectGraph returns a subject's Neo4j knowledge-graph subtree --
// backs the frontend graph visualization.
func (s *Service) GetSubjectGraph(ctx context.Context, subjectCode string, includeClos bool) (*dto.SubjectGraph, error) {
	return s.repo.GetSubjectGraph(ctx, subjectCode, includeClos)
}
