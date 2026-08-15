package service

import (
	"context"
	"errors"

	"github.com/google/uuid"

	"github.com/edugraph-ai/edugraph/internal/modeling/dto"
	"github.com/edugraph-ai/edugraph/internal/modeling/repository"
	apperrors "github.com/edugraph-ai/edugraph/pkg/errors"
)

type Service struct {
	repo *repository.Repository
}

func New(repo *repository.Repository) *Service {
	return &Service{repo: repo}
}

// ListCandidateSnapshots backs the governance review queue.
func (s *Service) ListCandidateSnapshots(ctx context.Context) ([]dto.ModelSnapshot, error) {
	snapshots, err := s.repo.ListCandidates(ctx)
	if err != nil {
		return nil, apperrors.Internal(err)
	}
	return snapshots, nil
}

// PromoteSnapshot activates a candidate refit (EG-GCKT spec section 19:
// "review and rollback mechanisms for high-impact graph changes" --
// promoting is the review half; the previous active snapshot survives as
// 'superseded', giving rollback by promoting it again).
func (s *Service) PromoteSnapshot(ctx context.Context, reviewerID, snapshotID uuid.UUID) (*dto.ModelSnapshot, error) {
	snapshot, err := s.repo.PromoteSnapshot(ctx, snapshotID, reviewerID)
	if errors.Is(err, repository.ErrSnapshotNotFound) {
		return nil, apperrors.NotFound("candidate model snapshot not found, or already reviewed")
	}
	if err != nil {
		return nil, apperrors.Internal(err)
	}
	return snapshot, nil
}

// RejectSnapshot declines a candidate refit -- it never becomes active.
func (s *Service) RejectSnapshot(ctx context.Context, reviewerID, snapshotID uuid.UUID) (*dto.ModelSnapshot, error) {
	snapshot, err := s.repo.RejectSnapshot(ctx, snapshotID, reviewerID)
	if errors.Is(err, repository.ErrSnapshotNotFound) {
		return nil, apperrors.NotFound("candidate model snapshot not found, or already reviewed")
	}
	if err != nil {
		return nil, apperrors.Internal(err)
	}
	return snapshot, nil
}
