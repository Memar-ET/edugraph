package service

import (
	"context"
	"errors"

	"github.com/edugraph-ai/edugraph/internal/curriculum/dto"
	"github.com/edugraph-ai/edugraph/internal/curriculum/repository"
	apperrors "github.com/edugraph-ai/edugraph/pkg/errors"
	"github.com/google/uuid"
)

// AddTopicPrerequisite validates and records a "topic requires
// prerequisite" edge, then best-effort mirrors it into Neo4j (the graph
// the gap-analysis root-cause walk and study-plan topological sort
// traverse). Postgres commit is the success criterion; a graph failure is
// reported in the response, not an error -- matching Approve's contract.
func (s *Service) AddTopicPrerequisite(
	ctx context.Context, userID, topicID uuid.UUID, req dto.AddPrerequisiteRequest,
) (*dto.AddPrerequisiteResponse, error) {
	prereqID, err := uuid.Parse(req.PrerequisiteTopicID)
	if err != nil {
		return nil, apperrors.BadRequest("invalid prerequisiteTopicId")
	}
	if prereqID == topicID {
		return nil, apperrors.BadRequest("a topic cannot be its own prerequisite")
	}
	weight := 1.0
	if req.Weight != nil {
		weight = *req.Weight
	}

	link, err := s.repo.AddTopicPrerequisite(ctx, topicID, prereqID, weight, userID)
	if errors.Is(err, repository.ErrTopicNotFound) {
		return nil, apperrors.NotFound("topic or prerequisite topic not found")
	}
	if errors.Is(err, repository.ErrPrerequisiteCycle) {
		return nil, apperrors.Conflict("this link would create a prerequisite cycle")
	}
	if err != nil {
		return nil, apperrors.Internal(err)
	}

	resp := &dto.AddPrerequisiteResponse{Link: *link, GraphSynced: true}
	if err := s.repo.SyncPrerequisiteToNeo4j(ctx, topicID, prereqID, weight); err != nil {
		resp.GraphSynced = false
		resp.GraphError = err.Error()
	}
	return resp, nil
}

// ListTopicPrerequisites returns a topic's direct prerequisites.
func (s *Service) ListTopicPrerequisites(ctx context.Context, topicID uuid.UUID) ([]dto.PrerequisiteLink, error) {
	links, err := s.repo.ListTopicPrerequisites(ctx, topicID)
	if err != nil {
		return nil, apperrors.Internal(err)
	}
	return links, nil
}
