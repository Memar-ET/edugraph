package service

import (
	"context"

	"github.com/edugraph-ai/edugraph/internal/curriculum/dto"
	apperrors "github.com/edugraph-ai/edugraph/pkg/errors"
)

// QMatrixQuality backs the curriculum-officer-facing Q-matrix quality
// report (EG-GCKT checklist sections 5/16).
func (s *Service) QMatrixQuality(ctx context.Context, subjectCode string) (*dto.QMatrixQualityReport, error) {
	report, err := s.repo.QMatrixQuality(ctx, subjectCode)
	if err != nil {
		return nil, apperrors.Internal(err)
	}
	return report, nil
}

// PrerequisiteQuality backs the prerequisite-graph structural validation
// report (EG-GCKT checklist section 4).
func (s *Service) PrerequisiteQuality(ctx context.Context, subjectCode string) (*dto.PrerequisiteQualityReport, error) {
	report, err := s.repo.PrerequisiteQuality(ctx, subjectCode)
	if err != nil {
		return nil, apperrors.Internal(err)
	}
	return report, nil
}
