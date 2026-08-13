package service

import (
	"context"
	"errors"

	"github.com/edugraph-ai/edugraph/internal/teacher/dto"
	"github.com/edugraph-ai/edugraph/internal/teacher/repository"
	apperrors "github.com/edugraph-ai/edugraph/pkg/errors"
	"github.com/edugraph-ai/edugraph/pkg/pagination"
)

type Service struct {
	repo *repository.Repository
}

func New(repo *repository.Repository) *Service {
	return &Service{repo: repo}
}

func (s *Service) Create(ctx context.Context, req dto.CreateTeacherRequest) (dto.TeacherResponse, error) {
	t, err := s.repo.Create(ctx, repository.CreateTeacherParams{
		UserID:           req.UserID,
		SchoolID:         req.SchoolID,
		SubjectSpecialty: nilIfEmpty(req.SubjectSpecialty),
	})
	if err != nil {
		return dto.TeacherResponse{}, apperrors.Internal(err)
	}
	return toResponse(t), nil
}

// Get and List apply the same caller-scoping policy as
// student/service.go's Get/List -- see there for the full rationale.
func (s *Service) Get(ctx context.Context, userID, role, id string) (dto.TeacherResponse, error) {
	t, err := s.repo.GetByID(ctx, id)
	if errors.Is(err, repository.ErrNotFound) {
		return dto.TeacherResponse{}, apperrors.NotFound("teacher not found")
	}
	if err != nil {
		return dto.TeacherResponse{}, apperrors.Internal(err)
	}

	if err := s.authorizeScope(ctx, userID, role, t.SchoolID); err != nil {
		return dto.TeacherResponse{}, err
	}

	return toResponse(t), nil
}

func (s *Service) List(ctx context.Context, userID, role, requestedSchoolID string, p pagination.Params) ([]dto.TeacherResponse, int64, error) {
	schoolID, regionID, err := s.scopeFilters(ctx, userID, role, requestedSchoolID)
	if err != nil {
		return nil, 0, err
	}

	teachers, total, err := s.repo.List(ctx, schoolID, regionID, p.Limit, p.Offset())
	if err != nil {
		return nil, 0, apperrors.Internal(err)
	}
	resp := make([]dto.TeacherResponse, 0, len(teachers))
	for _, t := range teachers {
		resp = append(resp, toResponse(t))
	}
	return resp, total, nil
}

func (s *Service) authorizeScope(ctx context.Context, userID, role, recordSchoolID string) error {
	switch role {
	case "ministry_admin":
		return nil
	case "regional_admin":
		_, ownRegion, err := s.repo.CallerScope(ctx, userID)
		if err != nil {
			return scopeErr(err)
		}
		recordRegion, err := s.repo.SchoolRegionID(ctx, recordSchoolID)
		if err != nil {
			return scopeErr(err)
		}
		if ownRegion == "" || recordRegion != ownRegion {
			return apperrors.NotFound("teacher not found")
		}
		return nil
	case "school_admin", "teacher":
		ownSchool, _, err := s.repo.CallerScope(ctx, userID)
		if err != nil {
			return scopeErr(err)
		}
		if ownSchool == "" || recordSchoolID != ownSchool {
			return apperrors.NotFound("teacher not found")
		}
		return nil
	default:
		return apperrors.Forbidden("not permitted to view teacher records")
	}
}

func (s *Service) scopeFilters(ctx context.Context, userID, role, requestedSchoolID string) (schoolID, regionID string, err error) {
	switch role {
	case "ministry_admin":
		return requestedSchoolID, "", nil
	case "regional_admin":
		_, ownRegion, err := s.repo.CallerScope(ctx, userID)
		if err != nil {
			return "", "", scopeErr(err)
		}
		if ownRegion == "" {
			return "", "", apperrors.Forbidden("regional admin has no region assigned")
		}
		return requestedSchoolID, ownRegion, nil
	case "school_admin", "teacher":
		ownSchool, _, err := s.repo.CallerScope(ctx, userID)
		if err != nil {
			return "", "", scopeErr(err)
		}
		if ownSchool == "" {
			return "", "", apperrors.Forbidden("no school assigned to this account")
		}
		return ownSchool, "", nil
	default:
		return "", "", apperrors.Forbidden("not permitted to view teacher records")
	}
}

func scopeErr(err error) error {
	if errors.Is(err, repository.ErrNotFound) {
		return apperrors.NotFound("user not found")
	}
	return apperrors.Internal(err)
}

func (s *Service) Update(ctx context.Context, id string, req dto.UpdateTeacherRequest) (dto.TeacherResponse, error) {
	t, err := s.repo.Update(ctx, id, nilIfEmpty(req.SubjectSpecialty))
	if errors.Is(err, repository.ErrNotFound) {
		return dto.TeacherResponse{}, apperrors.NotFound("teacher not found")
	}
	if err != nil {
		return dto.TeacherResponse{}, apperrors.Internal(err)
	}
	return toResponse(t), nil
}

func (s *Service) Delete(ctx context.Context, id string) error {
	err := s.repo.Delete(ctx, id)
	if errors.Is(err, repository.ErrNotFound) {
		return apperrors.NotFound("teacher not found")
	}
	if err != nil {
		return apperrors.Internal(err)
	}
	return nil
}

func toResponse(t repository.Teacher) dto.TeacherResponse {
	resp := dto.TeacherResponse{ID: t.ID, UserID: t.UserID, SchoolID: t.SchoolID, CreatedAt: t.CreatedAt}
	if t.SubjectSpecialty != nil {
		resp.SubjectSpecialty = *t.SubjectSpecialty
	}
	return resp
}

func nilIfEmpty(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}
