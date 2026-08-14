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

// Get fetches one student, scoped to the caller: ministry_admin sees any
// student; regional_admin only within their own region; school_admin/
// teacher only within their own school. The school_id/region_id used for
// this check is always resolved server-side from the caller's own user
// row (repo.CallerScope), never trusted from the request -- see
// assessment/service/school_quality.go's GetSchoolQualityScores for the
// same pattern applied to schools.
func (s *Service) Get(ctx context.Context, userID, role, id string) (dto.StudentResponse, error) {
	st, err := s.repo.GetByID(ctx, id)
	if errors.Is(err, repository.ErrNotFound) {
		return dto.StudentResponse{}, apperrors.NotFound("student not found")
	}
	if err != nil {
		return dto.StudentResponse{}, apperrors.Internal(err)
	}

	if err := s.authorizeScope(ctx, userID, role, st.SchoolID); err != nil {
		return dto.StudentResponse{}, err
	}

	return toResponse(st), nil
}

// List: ministry_admin gets an unfiltered (or caller-requested) view;
// regional_admin is always constrained to their own region regardless of
// what schoolID they pass (see repository.List's AND semantics -- a
// schoolID outside their region simply matches nothing); school_admin/
// teacher have schoolID forced to their own, ignoring whatever the
// request asked for. Anything else is refused -- there's no legitimate
// reason for e.g. a plain student account to browse this roster; that
// role is additionally kept off this route entirely at the router.
func (s *Service) List(ctx context.Context, userID, role, requestedSchoolID string, p pagination.Params) ([]dto.StudentResponse, int64, error) {
	schoolID, regionID, err := s.scopeFilters(ctx, userID, role, requestedSchoolID)
	if err != nil {
		return nil, 0, err
	}

	students, total, err := s.repo.List(ctx, schoolID, regionID, p.Limit, p.Offset())
	if err != nil {
		return nil, 0, apperrors.Internal(err)
	}
	resp := make([]dto.StudentResponse, 0, len(students))
	for _, st := range students {
		resp = append(resp, toResponse(st))
	}
	return resp, total, nil
}

// authorizeScope checks a specific record's school against the caller's
// own scope, for single-record fetches (Get).
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
			return apperrors.NotFound("student not found")
		}
		return nil
	case "school_admin", "teacher":
		ownSchool, _, err := s.repo.CallerScope(ctx, userID)
		if err != nil {
			return scopeErr(err)
		}
		if ownSchool == "" || recordSchoolID != ownSchool {
			return apperrors.NotFound("student not found")
		}
		return nil
	default:
		return apperrors.Forbidden("not permitted to view student records")
	}
}

// scopeFilters resolves the (schoolID, regionID) filter pair to pass to
// repo.List for a given caller, per the role policy documented on List.
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
		return "", "", apperrors.Forbidden("not permitted to view student records")
	}
}

func scopeErr(err error) error {
	if errors.Is(err, repository.ErrNotFound) {
		return apperrors.NotFound("user not found")
	}
	return apperrors.Internal(err)
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
