package service

import (
	"context"
	"errors"

	"github.com/google/uuid"

	"github.com/edugraph-ai/edugraph/internal/curriculum/dto"
	"github.com/edugraph-ai/edugraph/internal/curriculum/repository"
	apperrors "github.com/edugraph-ai/edugraph/pkg/errors"
)

// AddItemSkillMapping validates and records a new versioned Q-matrix
// entry, then best-effort mirrors it into Neo4j -- mirrors
// AddTopicPrerequisite's contract exactly (Postgres commit is the
// success criterion, a graph failure is reported not erred).
func (s *Service) AddItemSkillMapping(
	ctx context.Context, userID, questionID uuid.UUID, req dto.AddItemSkillMappingRequest,
) (*dto.AddItemSkillMappingResponse, error) {
	topicID, err := uuid.Parse(req.TopicID)
	if err != nil {
		return nil, apperrors.BadRequest("invalid topicId")
	}

	mapping, err := s.repo.AddItemSkillMapping(ctx, repository.AddItemSkillMappingParams{
		QuestionID:       questionID,
		TopicID:          topicID,
		CloCode:          req.CloCode,
		Relevance:        req.Relevance,
		CognitiveLevel:   req.CognitiveLevel,
		GenerationMethod: req.GenerationMethod,
		UserID:           userID,
	})
	if errors.Is(err, repository.ErrTopicNotFound) {
		return nil, apperrors.NotFound("topic not found")
	}
	if errors.Is(err, repository.ErrQuestionNotFound) {
		return nil, apperrors.NotFound("question not found")
	}
	if err != nil {
		return nil, apperrors.Internal(err)
	}

	return s.syncItemSkillMappingAndRespond(ctx, *mapping), nil
}

// ListItemSkillMappings returns a question's current Q-matrix rows.
func (s *Service) ListItemSkillMappings(ctx context.Context, questionID uuid.UUID) ([]dto.ItemSkillMapping, error) {
	mappings, err := s.repo.ListItemSkillMappings(ctx, questionID)
	if err != nil {
		return nil, apperrors.Internal(err)
	}
	return mappings, nil
}

func (s *Service) syncItemSkillMappingAndRespond(ctx context.Context, mapping dto.ItemSkillMapping) *dto.AddItemSkillMappingResponse {
	mappingID := uuid.MustParse(mapping.ID)
	questionID := uuid.MustParse(mapping.QuestionID)
	topicID := uuid.MustParse(mapping.TopicID)

	resp := &dto.AddItemSkillMappingResponse{Mapping: mapping, GraphSynced: true}
	if err := s.repo.SyncItemSkillMappingToNeo4j(ctx, mappingID, questionID, topicID, mapping.Relevance, mapping.IsValidated); err != nil {
		resp.GraphSynced = false
		resp.GraphError = err.Error()
		return resp
	}
	if err := s.repo.MarkItemSkillMappingSynced(ctx, mappingID); err != nil {
		resp.GraphSynced = false
		resp.GraphError = err.Error()
	}
	return resp
}

// ResyncItemSkillMappingsToNeo4j re-mirrors every current Q-matrix row
// with neo4j_written = false -- the Q-matrix counterpart to
// ResyncPrerequisitesToNeo4j (feature 1.5's pattern).
func (s *Service) ResyncItemSkillMappingsToNeo4j(ctx context.Context) (*dto.ResyncItemSkillMappingsResponse, error) {
	pending, err := s.repo.ListUnsyncedItemSkillMappings(ctx)
	if err != nil {
		return nil, apperrors.Internal(err)
	}

	resp := &dto.ResyncItemSkillMappingsResponse{}
	for _, m := range pending {
		if err := s.repo.SyncItemSkillMappingToNeo4j(ctx, m.ID, m.QuestionID, m.TopicID, m.Relevance, m.IsValidated); err != nil {
			resp.Failed++
			continue
		}
		if err := s.repo.MarkItemSkillMappingSynced(ctx, m.ID); err != nil {
			resp.Failed++
			continue
		}
		resp.Synced++
	}
	return resp, nil
}
