package service

import (
	"context"
	"errors"

	"github.com/edugraph-ai/edugraph/internal/jobs/dto"
	"github.com/edugraph-ai/edugraph/internal/jobs/repository"
	apperrors "github.com/edugraph-ai/edugraph/pkg/errors"
	"github.com/edugraph-ai/edugraph/pkg/pagination"
)

type Service struct {
	repo *repository.Repository
}

func New(repo *repository.Repository) *Service {
	return &Service{repo: repo}
}

func (s *Service) Create(ctx context.Context, createdBy string, req dto.CreateJobRequest) (dto.JobResponse, error) {
	j, err := s.repo.Create(ctx, req.Type, req.Payload, &createdBy)
	if err != nil {
		return dto.JobResponse{}, apperrors.Internal(err)
	}
	return toResponse(j), nil
}

// Get is scoped to the caller's own job (checklist 11.3 finding: this
// used to take no userID at all, letting any authenticated caller read
// any job's payload/result by id -- List already correctly used
// ListMine/ListByCreator, this was the one path that didn't match it).
// A worker updating status (UpdateStatus) legitimately has no "creator"
// identity of its own, so that path is intentionally left unscoped --
// see its own doc comment.
func (s *Service) Get(ctx context.Context, userID, id string) (dto.JobResponse, error) {
	j, err := s.repo.GetByID(ctx, id)
	if errors.Is(err, repository.ErrNotFound) {
		return dto.JobResponse{}, apperrors.NotFound("job not found")
	}
	if err != nil {
		return dto.JobResponse{}, apperrors.Internal(err)
	}
	if j.CreatedBy == nil || *j.CreatedBy != userID {
		return dto.JobResponse{}, apperrors.NotFound("job not found")
	}
	return toResponse(j), nil
}

func (s *Service) ListMine(ctx context.Context, userID string, p pagination.Params) ([]dto.JobResponse, int64, error) {
	jobs, total, err := s.repo.ListByCreator(ctx, userID, p.Limit, p.Offset())
	if err != nil {
		return nil, 0, apperrors.Internal(err)
	}
	resp := make([]dto.JobResponse, 0, len(jobs))
	for _, j := range jobs {
		resp = append(resp, toResponse(j))
	}
	return resp, total, nil
}

// UpdateStatus is called by workers (e.g. the ai-service callback, or
// cmd/sync-worker) to report job progress.
func (s *Service) UpdateStatus(ctx context.Context, id string, req dto.UpdateJobStatusRequest) (dto.JobResponse, error) {
	var errMsg *string
	if req.Error != "" {
		errMsg = &req.Error
	}
	j, err := s.repo.UpdateStatus(ctx, id, req.Status, req.Result, errMsg)
	if errors.Is(err, repository.ErrNotFound) {
		return dto.JobResponse{}, apperrors.NotFound("job not found")
	}
	if err != nil {
		return dto.JobResponse{}, apperrors.Internal(err)
	}
	return toResponse(j), nil
}

func toResponse(j repository.Job) dto.JobResponse {
	resp := dto.JobResponse{ID: j.ID, Type: j.Type, Status: j.Status, Payload: j.Payload, Result: j.Result, CreatedAt: j.CreatedAt, UpdatedAt: j.UpdatedAt}
	if j.Error != nil {
		resp.Error = *j.Error
	}
	return resp
}
