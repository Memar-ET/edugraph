package service

import (
	"context"

	"github.com/edugraph-ai/edugraph/internal/ministry/dto"
	"github.com/edugraph-ai/edugraph/internal/ministry/repository"
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

func (s *Service) Overview(ctx context.Context) (dto.OverviewResponse, error) {
	o, err := s.repo.Overview(ctx)
	if err != nil {
		return dto.OverviewResponse{}, apperrors.Internal(err)
	}
	return dto.OverviewResponse{
		TotalRegions:  o.TotalRegions,
		TotalSchools:  o.TotalSchools,
		TotalStudents: o.TotalStudents,
		TotalTeachers: o.TotalTeachers,
	}, nil
}

func (s *Service) RegionStats(ctx context.Context, regionID string) (dto.RegionStatsResponse, error) {
	stats, err := s.repo.RegionStats(ctx, regionID)
	if err != nil {
		return dto.RegionStatsResponse{}, apperrors.Internal(err)
	}
	return dto.RegionStatsResponse{
		RegionID:           stats.RegionID,
		SchoolCount:        stats.SchoolCount,
		StudentCount:       stats.StudentCount,
		TeacherCount:       stats.TeacherCount,
		AvgAssessmentScore: stats.AvgAssessmentScore,
	}, nil
}
