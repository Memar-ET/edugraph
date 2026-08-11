package service

import (
	"context"
	"errors"

	"github.com/edugraph-ai/edugraph/internal/region/dto"
	"github.com/edugraph-ai/edugraph/internal/region/repository"
	apperrors "github.com/edugraph-ai/edugraph/pkg/errors"
	"github.com/edugraph-ai/edugraph/pkg/pagination"
)

type Service struct {
	repo *repository.Repository
}

func New(repo *repository.Repository) *Service {
	return &Service{repo: repo}
}

func (s *Service) Create(ctx context.Context, req dto.CreateRegionRequest) (dto.RegionResponse, error) {
	reg, err := s.repo.Create(ctx, req.Name, req.Code)
	if err != nil {
		return dto.RegionResponse{}, apperrors.Internal(err)
	}
	return toResponse(reg), nil
}

func (s *Service) Get(ctx context.Context, id string) (dto.RegionResponse, error) {
	reg, err := s.repo.GetByID(ctx, id)
	if errors.Is(err, repository.ErrNotFound) {
		return dto.RegionResponse{}, apperrors.NotFound("region not found")
	}
	if err != nil {
		return dto.RegionResponse{}, apperrors.Internal(err)
	}
	return toResponse(reg), nil
}

func (s *Service) List(ctx context.Context, p pagination.Params) ([]dto.RegionResponse, int64, error) {
	regions, total, err := s.repo.List(ctx, p.Limit, p.Offset())
	if err != nil {
		return nil, 0, apperrors.Internal(err)
	}
	resp := make([]dto.RegionResponse, 0, len(regions))
	for _, reg := range regions {
		resp = append(resp, toResponse(reg))
	}
	return resp, total, nil
}

func (s *Service) Update(ctx context.Context, id string, req dto.UpdateRegionRequest) (dto.RegionResponse, error) {
	reg, err := s.repo.Update(ctx, id, req.Name)
	if errors.Is(err, repository.ErrNotFound) {
		return dto.RegionResponse{}, apperrors.NotFound("region not found")
	}
	if err != nil {
		return dto.RegionResponse{}, apperrors.Internal(err)
	}
	return toResponse(reg), nil
}

func (s *Service) Delete(ctx context.Context, id string) error {
	err := s.repo.Delete(ctx, id)
	if errors.Is(err, repository.ErrNotFound) {
		return apperrors.NotFound("region not found")
	}
	if err != nil {
		return apperrors.Internal(err)
	}
	return nil
}

func toResponse(r repository.Region) dto.RegionResponse {
	return dto.RegionResponse{ID: r.ID, Name: r.Name, Code: r.Code, CreatedAt: r.CreatedAt}
}
