package service

import (
	"context"
	"errors"

	"github.com/edugraph-ai/edugraph/internal/assessment/dto"
	"github.com/edugraph-ai/edugraph/internal/assessment/repository"
	apperrors "github.com/edugraph-ai/edugraph/pkg/errors"
	"github.com/edugraph-ai/edugraph/pkg/pagination"
)

type Service struct {
	repo *repository.Repository
}

func New(repo *repository.Repository) *Service {
	return &Service{repo: repo}
}

func (s *Service) CreateAssessment(ctx context.Context, createdBy string, req dto.CreateAssessmentRequest) (dto.AssessmentResponse, error) {
	a, err := s.repo.CreateAssessment(ctx, repository.CreateAssessmentParams{
		SubjectID:  req.SubjectID,
		SchoolID:   nilIfEmpty(req.SchoolID),
		CreatedBy:  createdBy,
		Title:      req.Title,
		Type:       req.Type,
		TotalMarks: req.TotalMarks,
	})
	if err != nil {
		return dto.AssessmentResponse{}, apperrors.Internal(err)
	}
	return toAssessmentResponse(a), nil
}

func (s *Service) GetAssessment(ctx context.Context, id string) (dto.AssessmentResponse, error) {
	a, err := s.repo.GetAssessment(ctx, id)
	if errors.Is(err, repository.ErrNotFound) {
		return dto.AssessmentResponse{}, apperrors.NotFound("assessment not found")
	}
	if err != nil {
		return dto.AssessmentResponse{}, apperrors.Internal(err)
	}
	return toAssessmentResponse(a), nil
}

func (s *Service) ListAssessments(ctx context.Context, subjectID, schoolID string, p pagination.Params) ([]dto.AssessmentResponse, int64, error) {
	items, total, err := s.repo.ListAssessments(ctx, subjectID, schoolID, p.Limit, p.Offset())
	if err != nil {
		return nil, 0, apperrors.Internal(err)
	}
	resp := make([]dto.AssessmentResponse, 0, len(items))
	for _, a := range items {
		resp = append(resp, toAssessmentResponse(a))
	}
	return resp, total, nil
}

func (s *Service) AddQuestion(ctx context.Context, assessmentID string, req dto.CreateQuestionRequest) (dto.QuestionResponse, error) {
	q, err := s.repo.AddQuestion(ctx, repository.CreateQuestionParams{
		AssessmentID:  assessmentID,
		QuestionText:  req.QuestionText,
		Options:       req.Options,
		CorrectAnswer: req.CorrectAnswer,
		Marks:         req.Marks,
		OrderIndex:    req.OrderIndex,
	})
	if err != nil {
		return dto.QuestionResponse{}, apperrors.Internal(err)
	}
	return toQuestionResponse(q), nil
}

func (s *Service) ListQuestions(ctx context.Context, assessmentID string) ([]dto.QuestionResponse, error) {
	questions, err := s.repo.ListQuestions(ctx, assessmentID)
	if err != nil {
		return nil, apperrors.Internal(err)
	}
	resp := make([]dto.QuestionResponse, 0, len(questions))
	for _, q := range questions {
		resp = append(resp, toQuestionResponse(q))
	}
	return resp, nil
}

// SubmitResult auto-grades a submission by comparing each answer against the
// stored correct_answer and summing marks for correct answers.
func (s *Service) SubmitResult(ctx context.Context, assessmentID string, req dto.SubmitResultRequest) (dto.ResultResponse, error) {
	questions, err := s.repo.ListQuestions(ctx, assessmentID)
	if err != nil {
		return dto.ResultResponse{}, apperrors.Internal(err)
	}

	var score float64
	for _, q := range questions {
		if answer, ok := req.Answers[q.ID]; ok && answer == q.CorrectAnswer {
			score += float64(q.Marks)
		}
	}

	result, err := s.repo.SubmitResult(ctx, assessmentID, req.StudentID, req.Answers, score)
	if err != nil {
		return dto.ResultResponse{}, apperrors.Internal(err)
	}
	return toResultResponse(result), nil
}

func (s *Service) ListResultsByStudent(ctx context.Context, studentID string) ([]dto.ResultResponse, error) {
	results, err := s.repo.ListResultsByStudent(ctx, studentID)
	if err != nil {
		return nil, apperrors.Internal(err)
	}
	resp := make([]dto.ResultResponse, 0, len(results))
	for _, res := range results {
		resp = append(resp, toResultResponse(res))
	}
	return resp, nil
}

func (s *Service) ListResultsByAssessment(ctx context.Context, assessmentID string) ([]dto.ResultResponse, error) {
	results, err := s.repo.ListResultsByAssessment(ctx, assessmentID)
	if err != nil {
		return nil, apperrors.Internal(err)
	}
	resp := make([]dto.ResultResponse, 0, len(results))
	for _, res := range results {
		resp = append(resp, toResultResponse(res))
	}
	return resp, nil
}

func toAssessmentResponse(a repository.Assessment) dto.AssessmentResponse {
	resp := dto.AssessmentResponse{ID: a.ID, SubjectID: a.SubjectID, Title: a.Title, Type: a.Type, TotalMarks: a.TotalMarks, CreatedAt: a.CreatedAt}
	if a.SchoolID != nil {
		resp.SchoolID = *a.SchoolID
	}
	return resp
}

func toQuestionResponse(q repository.Question) dto.QuestionResponse {
	return dto.QuestionResponse{ID: q.ID, AssessmentID: q.AssessmentID, QuestionText: q.QuestionText, Options: q.Options, Marks: q.Marks, OrderIndex: q.OrderIndex}
}

func toResultResponse(r repository.Result) dto.ResultResponse {
	return dto.ResultResponse{ID: r.ID, AssessmentID: r.AssessmentID, StudentID: r.StudentID, Score: r.Score, SubmittedAt: r.SubmittedAt, GradedAt: r.GradedAt}
}

func nilIfEmpty(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}
