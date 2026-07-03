package service

import (
	"context"

	"github.com/edugraph-ai/edugraph/internal/career/dto"
	"github.com/edugraph-ai/edugraph/internal/career/repository"
	"github.com/edugraph-ai/edugraph/pkg/ai"
	apperrors "github.com/edugraph-ai/edugraph/pkg/errors"
)

type Service struct {
	repo *repository.Repository
	ai   *ai.Client
}

func New(repo *repository.Repository, aiClient *ai.Client) *Service {
	return &Service{repo: repo, ai: aiClient}
}

func (s *Service) Create(ctx context.Context, req dto.CreateCareerPathRequest) (dto.CareerPathResponse, error) {
	cp, err := s.repo.Create(ctx, req.Title, nilIfEmpty(req.Description), req.RequiredSubjects)
	if err != nil {
		return dto.CareerPathResponse{}, apperrors.Internal(err)
	}
	return toResponse(cp), nil
}

func (s *Service) List(ctx context.Context) ([]dto.CareerPathResponse, error) {
	paths, err := s.repo.List(ctx)
	if err != nil {
		return nil, apperrors.Internal(err)
	}
	resp := make([]dto.CareerPathResponse, 0, len(paths))
	for _, cp := range paths {
		resp = append(resp, toResponse(cp))
	}
	return resp, nil
}

// GenerateMatches asks the ai-service to score every career path against the
// student's per-subject performance, then persists the results.
func (s *Service) GenerateMatches(ctx context.Context, studentID string) ([]dto.MatchResponse, error) {
	grades, err := s.repo.StudentSubjectAverages(ctx, studentID)
	if err != nil {
		return nil, apperrors.Internal(err)
	}
	if len(grades) == 0 {
		return nil, apperrors.BadRequest("student has no graded assessments to base a career match on")
	}

	results, err := s.ai.MatchCareers(ctx, ai.CareerMatchRequest{StudentID: studentID, Grades: grades})
	if err != nil {
		return nil, apperrors.Wrap(err, 502, "ai_service_unavailable", "career matching engine is unavailable")
	}

	matches := make([]repository.Match, 0, len(results))
	for _, res := range results {
		matches = append(matches, repository.Match{CareerPathID: res.CareerPathID, Score: res.Score})
	}
	if err := s.repo.SaveMatches(ctx, studentID, matches); err != nil {
		return nil, apperrors.Internal(err)
	}

	return s.Matches(ctx, studentID)
}

func (s *Service) Matches(ctx context.Context, studentID string) ([]dto.MatchResponse, error) {
	matches, err := s.repo.ListMatches(ctx, studentID)
	if err != nil {
		return nil, apperrors.Internal(err)
	}
	resp := make([]dto.MatchResponse, 0, len(matches))
	for _, m := range matches {
		resp = append(resp, dto.MatchResponse{CareerPathID: m.CareerPathID, Title: m.Title, Score: m.Score})
	}
	return resp, nil
}

func toResponse(cp repository.CareerPath) dto.CareerPathResponse {
	resp := dto.CareerPathResponse{ID: cp.ID, Title: cp.Title, RequiredSubjects: cp.RequiredSubjects, CreatedAt: cp.CreatedAt}
	if cp.Description != nil {
		resp.Description = *cp.Description
	}
	return resp
}

func nilIfEmpty(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}
