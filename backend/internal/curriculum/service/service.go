package service

import (
	"context"
	"errors"

	"github.com/edugraph-ai/edugraph/internal/curriculum/dto"
	"github.com/edugraph-ai/edugraph/internal/curriculum/repository"
	apperrors "github.com/edugraph-ai/edugraph/pkg/errors"
)

type Service struct {
	repo *repository.Repository
}

func New(repo *repository.Repository) *Service {
	return &Service{repo: repo}
}

func (s *Service) CreateSubject(ctx context.Context, req dto.CreateSubjectRequest) (dto.SubjectResponse, error) {
	subj, err := s.repo.CreateSubject(ctx, req.Name, req.Code, req.GradeLevel)
	if err != nil {
		return dto.SubjectResponse{}, apperrors.Internal(err)
	}
	return dto.SubjectResponse{ID: subj.ID, Name: subj.Name, Code: subj.Code, GradeLevel: subj.GradeLevel, CreatedAt: subj.CreatedAt}, nil
}

func (s *Service) ListSubjects(ctx context.Context, gradeLevel int16) ([]dto.SubjectResponse, error) {
	subjects, err := s.repo.ListSubjects(ctx, gradeLevel)
	if err != nil {
		return nil, apperrors.Internal(err)
	}
	resp := make([]dto.SubjectResponse, 0, len(subjects))
	for _, subj := range subjects {
		resp = append(resp, dto.SubjectResponse{ID: subj.ID, Name: subj.Name, Code: subj.Code, GradeLevel: subj.GradeLevel, CreatedAt: subj.CreatedAt})
	}
	return resp, nil
}

func (s *Service) CreateUnit(ctx context.Context, req dto.CreateUnitRequest) (dto.UnitResponse, error) {
	u, err := s.repo.CreateUnit(ctx, repository.CreateUnitParams{
		SubjectID:   req.SubjectID,
		Title:       req.Title,
		Description: nilIfEmpty(req.Description),
		OrderIndex:  req.OrderIndex,
	})
	if err != nil {
		return dto.UnitResponse{}, apperrors.Internal(err)
	}
	return toUnitResponse(u), nil
}

func (s *Service) GetUnit(ctx context.Context, id string) (dto.UnitResponse, error) {
	u, err := s.repo.GetUnit(ctx, id)
	if errors.Is(err, repository.ErrNotFound) {
		return dto.UnitResponse{}, apperrors.NotFound("curriculum unit not found")
	}
	if err != nil {
		return dto.UnitResponse{}, apperrors.Internal(err)
	}
	return toUnitResponse(u), nil
}

func (s *Service) ListUnits(ctx context.Context, subjectID string) ([]dto.UnitResponse, error) {
	units, err := s.repo.ListUnits(ctx, subjectID)
	if err != nil {
		return nil, apperrors.Internal(err)
	}
	resp := make([]dto.UnitResponse, 0, len(units))
	for _, u := range units {
		resp = append(resp, toUnitResponse(u))
	}
	return resp, nil
}

func (s *Service) UpdateUnit(ctx context.Context, id string, req dto.UpdateUnitRequest) (dto.UnitResponse, error) {
	u, err := s.repo.UpdateUnit(ctx, id, req.Title, nilIfEmpty(req.Description), req.OrderIndex)
	if errors.Is(err, repository.ErrNotFound) {
		return dto.UnitResponse{}, apperrors.NotFound("curriculum unit not found")
	}
	if err != nil {
		return dto.UnitResponse{}, apperrors.Internal(err)
	}
	return toUnitResponse(u), nil
}

func (s *Service) DeleteUnit(ctx context.Context, id string) error {
	err := s.repo.DeleteUnit(ctx, id)
	if errors.Is(err, repository.ErrNotFound) {
		return apperrors.NotFound("curriculum unit not found")
	}
	if err != nil {
		return apperrors.Internal(err)
	}
	return nil
}

func (s *Service) AddPrerequisite(ctx context.Context, unitID string, req dto.AddPrerequisiteRequest) error {
	if err := s.repo.AddPrerequisite(ctx, unitID, req.PrerequisiteUnitID); err != nil {
		return apperrors.Internal(err)
	}
	return nil
}

func (s *Service) Prerequisites(ctx context.Context, unitID string) ([]dto.PrerequisiteResponse, error) {
	nodes, err := s.repo.Prerequisites(ctx, unitID)
	if err != nil {
		return nil, apperrors.Internal(err)
	}
	resp := make([]dto.PrerequisiteResponse, 0, len(nodes))
	for _, n := range nodes {
		resp = append(resp, dto.PrerequisiteResponse{UnitID: n.ID, Title: n.Title})
	}
	return resp, nil
}

func toUnitResponse(u repository.Unit) dto.UnitResponse {
	resp := dto.UnitResponse{ID: u.ID, SubjectID: u.SubjectID, Title: u.Title, OrderIndex: u.OrderIndex, CreatedAt: u.CreatedAt}
	if u.Description != nil {
		resp.Description = *u.Description
	}
	return resp
}

func nilIfEmpty(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}
