package service

import (
	"context"
	"time"

	"github.com/edugraph-ai/edugraph/internal/sync/dto"
	"github.com/edugraph-ai/edugraph/internal/sync/repository"
	apperrors "github.com/edugraph-ai/edugraph/pkg/errors"
)

type Service struct {
	repo *repository.Repository
}

func New(repo *repository.Repository) *Service {
	return &Service{repo: repo}
}

// Push accepts a batch of offline changes from a School Box device and
// queues them for the sync-worker to apply.
func (s *Service) Push(ctx context.Context, req dto.PushRequest) (dto.PushResponse, error) {
	for _, change := range req.Changes {
		_, err := s.repo.CreateLog(ctx, repository.CreateLogParams{
			DeviceID:   req.DeviceID,
			SchoolID:   req.SchoolID,
			EntityType: change.EntityType,
			EntityID:   change.EntityID,
			Operation:  change.Operation,
			Payload:    change.Payload,
		})
		if err != nil {
			return dto.PushResponse{}, apperrors.Internal(err)
		}
	}
	return dto.PushResponse{Accepted: len(req.Changes)}, nil
}

// Pull returns changes applied to schoolID after `since`, for a device to
// merge into its local offline store.
func (s *Service) Pull(ctx context.Context, schoolID string, since time.Time) (dto.PullResponse, error) {
	logs, err := s.repo.ListApplied(ctx, schoolID, since)
	if err != nil {
		return dto.PullResponse{}, apperrors.Internal(err)
	}

	changes := make([]dto.PulledChange, 0, len(logs))
	for _, l := range logs {
		pc := dto.PulledChange{EntityType: l.EntityType, EntityID: l.EntityID, Operation: l.Operation, Payload: l.Payload}
		if l.SyncedAt != nil {
			pc.SyncedAt = *l.SyncedAt
		}
		changes = append(changes, pc)
	}

	return dto.PullResponse{Changes: changes, ServerTime: time.Now()}, nil
}
