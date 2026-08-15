package service

import (
	"context"

	"github.com/google/uuid"

	"github.com/edugraph-ai/edugraph/internal/modeling/dto"
	apperrors "github.com/edugraph-ai/edugraph/pkg/errors"
)

// ListSkillStateSnapshots returns the historical snapshot record for one
// (student, topic) pair -- same ownership rules as Explain (own record
// for a student, same-school for a teacher/school_admin, unrestricted
// for ministry/regional/curriculum roles), since it's the same class of
// sensitive per-student data.
func (s *Service) ListSkillStateSnapshots(ctx context.Context, callerID, studentID, topicID uuid.UUID) ([]dto.SkillStateSnapshot, error) {
	if err := s.authorizeExplain(ctx, callerID, studentID); err != nil {
		return nil, err
	}

	snapshots, err := s.repo.ListSkillStateSnapshots(ctx, studentID, topicID)
	if err != nil {
		return nil, apperrors.Internal(err)
	}
	return snapshots, nil
}
