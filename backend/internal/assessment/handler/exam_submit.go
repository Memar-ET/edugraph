package handler

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/edugraph-ai/edugraph/internal/assessment/dto"
	apperrors "github.com/edugraph-ai/edugraph/pkg/errors"
	"github.com/edugraph-ai/edugraph/pkg/middleware"
	"github.com/edugraph-ai/edugraph/pkg/validator"
)

// ListExamQuestions handles GET /api/v1/exams/:id/questions -- what the
// exam-taking page fetches before the student answers anything (role
// student). Distinct from the teacher-only GetExam (which includes
// question counts/validation reports, not the question text).
func (h *Handler) ListExamQuestions(w http.ResponseWriter, r *http.Request) {
	examID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		middleware.WriteError(w, apperrors.BadRequest("invalid exam id"))
		return
	}

	userIDStr := middleware.UserID(r.Context())
	if userIDStr == "" {
		middleware.WriteError(w, apperrors.Unauthorized("user not authenticated"))
		return
	}
	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		middleware.WriteError(w, apperrors.Internal(err))
		return
	}

	questions, err := h.svc.ListExamQuestionsForStudent(r.Context(), userID, examID)
	if err != nil {
		middleware.WriteError(w, err)
		return
	}

	middleware.WriteJSON(w, http.StatusOK, questions)
}

// ListQuestionsForGrading handles GET /api/v1/exams/:id/grading-questions --
// backs the teacher "Grade Exam" spreadsheet's column headers (role
// teacher/school_admin). Includes answer_key, unlike ListExamQuestions.
func (h *Handler) ListQuestionsForGrading(w http.ResponseWriter, r *http.Request) {
	examID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		middleware.WriteError(w, apperrors.BadRequest("invalid exam id"))
		return
	}
	userID, err := callerID(r)
	if err != nil {
		middleware.WriteError(w, apperrors.Internal(err))
		return
	}

	questions, err := h.svc.ListQuestionsForGrading(r.Context(), userID, examID)
	if err != nil {
		middleware.WriteError(w, err)
		return
	}

	middleware.WriteJSON(w, http.StatusOK, questions)
}

// SubmitExam handles POST /api/v1/exams/:id/submit (Flow 1, digital --
// role student, enforced by middleware.RequireRole in the router).
func (h *Handler) SubmitExam(w http.ResponseWriter, r *http.Request) {
	examID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		middleware.WriteError(w, apperrors.BadRequest("invalid exam id"))
		return
	}

	userIDStr := middleware.UserID(r.Context())
	if userIDStr == "" {
		middleware.WriteError(w, apperrors.Unauthorized("user not authenticated"))
		return
	}
	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		middleware.WriteError(w, apperrors.Internal(err))
		return
	}

	var req dto.SubmitExamRequest
	if !decode(w, r, &req) {
		return
	}
	if err := validator.Struct(req); err != nil {
		middleware.WriteError(w, apperrors.BadRequest(err.Error()))
		return
	}

	resp, err := h.svc.SubmitExam(r.Context(), userID, examID, req)
	if err != nil {
		middleware.WriteError(w, err)
		return
	}

	middleware.WriteJSON(w, http.StatusOK, resp)
}

// BulkGradeExam handles POST /api/v1/exams/:id/grades/bulk (Flow 2,
// paper/teacher-encoded -- role teacher/school_admin).
func (h *Handler) BulkGradeExam(w http.ResponseWriter, r *http.Request) {
	examID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		middleware.WriteError(w, apperrors.BadRequest("invalid exam id"))
		return
	}

	var req dto.BulkGradeRequest
	if !decode(w, r, &req) {
		return
	}
	if err := validator.Struct(req); err != nil {
		middleware.WriteError(w, apperrors.BadRequest(err.Error()))
		return
	}

	gradedBy, err := uuid.Parse(middleware.UserID(r.Context()))
	if err != nil {
		middleware.WriteError(w, apperrors.Internal(err))
		return
	}

	resp, err := h.svc.BulkGradeExam(r.Context(), examID, gradedBy, req)
	if err != nil {
		middleware.WriteError(w, err)
		return
	}

	middleware.WriteJSON(w, http.StatusOK, resp)
}
