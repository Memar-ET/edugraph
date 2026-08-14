package service

import (
	"context"
	"errors"

	"github.com/edugraph-ai/edugraph/internal/school/dto"
	"github.com/edugraph-ai/edugraph/internal/school/repository"
	apperrors "github.com/edugraph-ai/edugraph/pkg/errors"
	"github.com/edugraph-ai/edugraph/pkg/pagination"
)

type Service struct {
	repo *repository.Repository
}

func New(repo *repository.Repository) *Service {
	return &Service{repo: repo}
}

func (s *Service) Create(ctx context.Context, req dto.CreateSchoolRequest) (dto.SchoolResponse, error) {
	sch, err := s.repo.Create(ctx, repository.CreateSchoolParams{
		RegionID: req.RegionID,
		Name:     req.Name,
		Code:     req.Code,
		Address:  nilIfEmpty(req.Address),
	})
	if err != nil {
		return dto.SchoolResponse{}, apperrors.Internal(err)
	}
	return toResponse(sch), nil
}

func (s *Service) Get(ctx context.Context, id string) (dto.SchoolResponse, error) {
	sch, err := s.repo.GetByID(ctx, id)
	if errors.Is(err, repository.ErrNotFound) {
		return dto.SchoolResponse{}, apperrors.NotFound("school not found")
	}
	if err != nil {
		return dto.SchoolResponse{}, apperrors.Internal(err)
	}
	return toResponse(sch), nil
}

// List forces regionID to the caller's own region for regional_admin,
// ignoring whatever the request asked for -- a regional_admin browsing
// another region's school directory shouldn't be possible even though
// school name/address isn't sensitive PII on its own (see
// docs/architecture/data-integrity.md). Every other role's requested
// regionID passes through unchanged (ministry_admin: national scope by
// design; school_admin/teacher: schools is low-sensitivity directory
// data, not narrowed further here).
func (s *Service) List(ctx context.Context, userID, role, regionID string, p pagination.Params) ([]dto.SchoolResponse, int64, error) {
	if role == "regional_admin" {
		own, err := s.repo.CallerRegionID(ctx, userID)
		if err != nil {
			if errors.Is(err, repository.ErrNotFound) {
				return nil, 0, apperrors.NotFound("user not found")
			}
			return nil, 0, apperrors.Internal(err)
		}
		if own == "" {
			return nil, 0, apperrors.Forbidden("regional admin has no region assigned")
		}
		regionID = own
	}

	schools, total, err := s.repo.List(ctx, regionID, p.Limit, p.Offset())
	if err != nil {
		return nil, 0, apperrors.Internal(err)
	}
	resp := make([]dto.SchoolResponse, 0, len(schools))
	for _, sch := range schools {
		resp = append(resp, toResponse(sch))
	}
	return resp, total, nil
}

func (s *Service) Update(ctx context.Context, id string, req dto.UpdateSchoolRequest) (dto.SchoolResponse, error) {
	sch, err := s.repo.Update(ctx, id, req.Name, nilIfEmpty(req.Address))
	if errors.Is(err, repository.ErrNotFound) {
		return dto.SchoolResponse{}, apperrors.NotFound("school not found")
	}
	if err != nil {
		return dto.SchoolResponse{}, apperrors.Internal(err)
	}
	return toResponse(sch), nil
}

func (s *Service) Delete(ctx context.Context, id string) error {
	err := s.repo.Delete(ctx, id)
	if errors.Is(err, repository.ErrNotFound) {
		return apperrors.NotFound("school not found")
	}
	if err != nil {
		return apperrors.Internal(err)
	}
	return nil
}

func toResponse(s repository.School) dto.SchoolResponse {
	resp := dto.SchoolResponse{ID: s.ID, RegionID: s.RegionID, Name: s.Name, Code: s.Code, CreatedAt: s.CreatedAt}
	if s.Address != nil {
		resp.Address = *s.Address
	}
	return resp
}

func nilIfEmpty(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}
