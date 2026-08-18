package service

import (
	"context"

	"github.com/google/uuid"

	"github.com/edugraph-ai/edugraph/internal/modeling/dto"
	apperrors "github.com/edugraph-ai/edugraph/pkg/errors"
)

// MySkillStates returns the authenticated student's own full skill map --
// GET /students/me/skill-states. callerID is the JWT subject; resolved to
// students.id server-side (never trusted from a URL param), the same
// pattern authorizeExplain uses for the {id}-scoped variant.
func (s *Service) MySkillStates(ctx context.Context, callerID uuid.UUID) (*dto.SkillStatesResponse, error) {
	auth, err := s.repo.FetchAuthContext(ctx, callerID)
	if err != nil {
		return nil, apperrors.Internal(err)
	}
	if auth.OwnStudentID == nil {
		return nil, apperrors.Forbidden("no student profile for this account")
	}

	states, err := s.repo.FetchAllSkillStates(ctx, *auth.OwnStudentID)
	if err != nil {
		return nil, apperrors.Internal(err)
	}

	return &dto.SkillStatesResponse{
		StudentID:   auth.OwnStudentID.String(),
		SkillStates: states,
	}, nil
}

// SkillStatesForStudent returns an arbitrary student's full skill map --
// GET /students/{id}/skill-states, the teacher/school_admin-facing
// counterpart to MySkillStates. studentID is caller-supplied, so this
// reuses authorizeExplain's server-side ownership check (own record for
// a student, same-school for a teacher/school_admin, unrestricted for
// ministry_admin/regional_admin/curriculum_officer) -- the same IDOR
// protection every other {id}-scoped student endpoint in this domain
// already has.
func (s *Service) SkillStatesForStudent(ctx context.Context, callerID, studentID uuid.UUID) (*dto.SkillStatesResponse, error) {
	if err := s.authorizeExplain(ctx, callerID, studentID); err != nil {
		return nil, err
	}

	states, err := s.repo.FetchAllSkillStates(ctx, studentID)
	if err != nil {
		return nil, apperrors.Internal(err)
	}

	return &dto.SkillStatesResponse{
		StudentID:   studentID.String(),
		SkillStates: states,
	}, nil
}
