package service

import (
	"context"
	"errors"

	"github.com/google/uuid"

	"github.com/edugraph-ai/edugraph/internal/curriculum/dto"
	"github.com/edugraph-ai/edugraph/internal/curriculum/repository"
	apperrors "github.com/edugraph-ai/edugraph/pkg/errors"
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

	inferMethod := req.InferMethod
	link, err := s.repo.AddTopicPrerequisite(ctx, topicID, prereqID, weight, inferMethod, userID)
	if errors.Is(err, repository.ErrTopicNotFound) {
		return nil, apperrors.NotFound("topic or prerequisite topic not found")
	}
	if errors.Is(err, repository.ErrPrerequisiteCycle) {
		return nil, apperrors.Conflict("this link would create a prerequisite cycle")
	}
	if err != nil {
		return nil, apperrors.Internal(err)
	}

	return s.syncPrerequisiteAndRespond(ctx, *link), nil
}

// ValidatePrerequisite confirms an existing "ai_inferred" (or any other)
// link -- the counterpart to AddTopicPrerequisite's inferMethod handling
// (feature 1.4). Re-syncs the link's isValidated property into Neo4j
// afterwards, same best-effort contract as creation.
func (s *Service) ValidatePrerequisite(ctx context.Context, userID, topicID, prereqID uuid.UUID) (*dto.AddPrerequisiteResponse, error) {
	link, err := s.repo.ValidatePrerequisite(ctx, topicID, prereqID, userID)
	if errors.Is(err, repository.ErrPrerequisiteNotFound) {
		return nil, apperrors.NotFound("prerequisite link not found")
	}
	if err != nil {
		return nil, apperrors.Internal(err)
	}

	return s.syncPrerequisiteAndRespond(ctx, *link), nil
}

// syncPrerequisiteAndRespond mirrors a just-written link into Neo4j and
// marks it synced on success -- shared by AddTopicPrerequisite and
// ValidatePrerequisite so both report the same GraphSynced contract.
func (s *Service) syncPrerequisiteAndRespond(ctx context.Context, link dto.PrerequisiteLink) *dto.AddPrerequisiteResponse {
	topicID := uuid.MustParse(link.TopicID)
	prereqID := uuid.MustParse(link.PrerequisiteTopicID)

	resp := &dto.AddPrerequisiteResponse{Link: link, GraphSynced: true}
	if err := s.repo.SyncPrerequisiteToNeo4j(ctx, topicID, prereqID, link.Weight, link.IsValidated, link.InferMethod); err != nil {
		resp.GraphSynced = false
		resp.GraphError = err.Error()
		return resp
	}
	if err := s.repo.MarkPrerequisiteSynced(ctx, topicID, prereqID); err != nil {
		resp.GraphSynced = false
		resp.GraphError = err.Error()
	}
	return resp
}

// ResyncPrerequisitesToNeo4j re-mirrors every prerequisite link with
// neo4j_written = false -- catches up links whose best-effort sync failed
// at write time, or that predate the neo4j_written column (V026) entirely
// (feature 1.5).
func (s *Service) ResyncPrerequisitesToNeo4j(ctx context.Context) (*dto.ResyncPrerequisitesResponse, error) {
	pending, err := s.repo.ListUnsyncedPrerequisites(ctx)
	if err != nil {
		return nil, apperrors.Internal(err)
	}

	resp := &dto.ResyncPrerequisitesResponse{}
	for _, p := range pending {
		if err := s.repo.SyncPrerequisiteToNeo4j(ctx, p.TopicID, p.PrereqID, p.Weight, p.IsValidated, p.InferMethod); err != nil {
			resp.Failed++
			continue
		}
		if err := s.repo.MarkPrerequisiteSynced(ctx, p.TopicID, p.PrereqID); err != nil {
			resp.Failed++
			continue
		}
		resp.Synced++
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
