package service

import (
	"context"

	"github.com/google/uuid"

	"github.com/edugraph-ai/edugraph/internal/assessment/dto"
	"github.com/edugraph-ai/edugraph/pkg/pagination"
)

// ListExams returns every exam at callerID's school, newest first --
// backs GET /api/v1/exams (TeacherExamListPage, ClassAnalyticsPage,
// QuestionBankPage, TeacherDashboardPage).
func (s *Service) ListExams(ctx context.Context, callerID uuid.UUID, p pagination.Params) ([]dto.ExamListItem, int64, error) {
	schoolID, err := s.repo.TeacherSchoolID(ctx, callerID)
	if err != nil {
		return nil, 0, err
	}
	return s.repo.ListExamsBySchool(ctx, schoolID, p)
}
