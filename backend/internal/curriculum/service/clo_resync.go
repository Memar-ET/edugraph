package service

import (
	"context"
	"fmt"

	"github.com/edugraph-ai/edugraph/internal/curriculum/dto"
	apperrors "github.com/edugraph-ai/edugraph/pkg/errors"
)

// ResyncCLOs re-mirrors all (:Topic)-[:HAS_CLO]->(:CLO) edges into Neo4j.
// It returns the count of successfully synced edges and any failure count.
func (s *Service) ResyncCLOs(ctx context.Context) (*dto.ResyncCLOsResult, error) {
	result, err := s.repo.ResyncCLOsToNeo4j(ctx)
	if err != nil {
		return nil, apperrors.Internal(fmt.Errorf("resync clos: %w", err))
	}
	return result, nil
}
