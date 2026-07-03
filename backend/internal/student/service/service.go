package service

import (
	"context"
	"errors"

	"github.com/edugraph-ai/edugraph/internal/student/dto"
	"github.com/edugraph-ai/edugraph/internal/student/repository"
	apperrors "github.com/edugraph-ai/edugraph/pkg/errors"
	"github.com/edugraph-ai/edugraph/pkg/pagination"
)

type Service struct {
	repo *repository.Repository
}

func New(repo *repository.Repository) *Service {
	return &Service{repo: repo}
}

func (s *Service) Create(ctx context.Context, req dto.CreateStudentRequest) (dto.StudentResponse, error) {
	st, err := s.repo.Create(ctx, repository.CreateStudentParams{
		UserID:      req.UserID,
		SchoolID:    req.SchoolID,
		AdmissionNo: req.AdmissionNo,
		GradeLevel:  req.GradeLevel,
	})
	if err != nil {
		return dto.StudentResponse{}, apperrors.Internal(err)
	}
	return toResponse(st), nil
}

func (s *Service) Get(ctx context.Context, id string) (dto.StudentResponse, error) {
	st, err := s.repo.GetByID(ctx, id)
	if errors.Is(err, repository.ErrNotFound) {
		return dto.StudentResponse{}, apperrors.NotFound("student not found")
	}
	if err != nil {
		return dto.StudentResponse{}, apperrors.Internal(err)
	}
	return toResponse(st), nil
}

func (s *Service) List(ctx context.Context, schoolID string, p pagination.Params) ([]dto.StudentResponse, int64, error) {
	students, total, err := s.repo.List(ctx, schoolID, p.Limit, p.Offset())
	if err != nil {
		return nil, 0, apperrors.Internal(err)
	}
	resp := make([]dto.StudentResponse, 0, len(students))
	for _, st := range students {
		resp = append(resp, toResponse(st))
	}
	return resp, total, nil
}

func (s *Service) UpdateGradeLevel(ctx context.Context, id string, req dto.UpdateStudentRequest) (dto.StudentResponse, error) {
	st, err := s.repo.UpdateGradeLevel(ctx, id, req.GradeLevel)
	if errors.Is(err, repository.ErrNotFound) {
		return dto.StudentResponse{}, apperrors.NotFound("student not found")
	}
	if err != nil {
		return dto.StudentResponse{}, apperrors.Internal(err)
	}
	return toResponse(st), nil
}

func (s *Service) Delete(ctx context.Context, id string) error {
	err := s.repo.Delete(ctx, id)
	if errors.Is(err, repository.ErrNotFound) {
		return apperrors.NotFound("student not found")
	}
	if err != nil {
		return apperrors.Internal(err)
	}
	return nil
}

func toResponse(st repository.Student) dto.StudentResponse {
	return dto.StudentResponse{
		ID: st.ID, UserID: st.UserID, SchoolID: st.SchoolID,
		AdmissionNo: st.AdmissionNo, GradeLevel: st.GradeLevel, CreatedAt: st.CreatedAt,
	}
}
