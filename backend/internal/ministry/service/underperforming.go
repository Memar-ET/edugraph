package service

import (
	"context"

	"github.com/edugraph-ai/edugraph/internal/ministry/dto"
	apperrors "github.com/edugraph-ai/edugraph/pkg/errors"
)

// UnderperformingSchools returns the worst schools in a region with their
// top weak topics. Returns an empty slice — never an error — when there is
// no gap data yet (cold-start regions).
func (s *Service) UnderperformingSchools(ctx context.Context, regionID string, limit int) (dto.UnderperformingResponse, error) {
	if limit <= 0 {
		limit = 10
	}

	schools, topicsBySchool, err := s.repo.UnderperformingSchools(ctx, regionID, limit)
	if err != nil {
		return dto.UnderperformingResponse{}, apperrors.Internal(err)
	}

	result := make([]dto.UnderperformingSchool, 0, len(schools))
	for _, sc := range schools {
		var weakTopics []dto.WeakTopic
		for _, t := range topicsBySchool[sc.SchoolID] {
			weakTopics = append(weakTopics, dto.WeakTopic{
				TopicID:          t.TopicID,
				TopicTitle:       t.TopicTitle,
				AvgMastery:       t.AvgMastery,
				AffectedStudents: t.AffectedStudents,
			})
		}
		if weakTopics == nil {
			weakTopics = []dto.WeakTopic{}
		}
		result = append(result, dto.UnderperformingSchool{
			SchoolID:           sc.SchoolID,
			SchoolName:         sc.SchoolName,
			SchoolCode:         sc.SchoolCode,
			QualityScore:       sc.QualityScore,
			MasteryRate:        sc.MasteryRate,
			FlaggedTopicsCount: sc.FlaggedTopicsCount,
			TopWeakTopics:      weakTopics,
		})
	}

	return dto.UnderperformingResponse{Schools: result}, nil
}
