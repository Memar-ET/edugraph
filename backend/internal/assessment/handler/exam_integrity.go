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

// ReportIntegrityEvents handles POST
// /api/v1/exams/{id}/attempts/current/events -- batched client-reported
// integrity signals for the caller's own attempt.
func (h *Handler) ReportIntegrityEvents(w http.ResponseWriter, r *http.Request) {
	examID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		middleware.WriteError(w, apperrors.BadRequest("invalid exam id"))
		return
	}
	userID, err := uuid.Parse(middleware.UserID(r.Context()))
	if err != nil {
		middleware.WriteError(w, apperrors.Unauthorized("user not authenticated"))
		return
	}
	var req dto.ReportIntegrityEventsRequest
	if !decode(w, r, &req) {
		return
	}
	if err := validator.Struct(req); err != nil {
		middleware.WriteError(w, apperrors.BadRequest(err.Error()))
		return
	}
	if err := h.svc.ReportIntegrityEvents(r.Context(), userID, examID, req); err != nil {
		middleware.WriteError(w, err)
		return
	}
	middleware.WriteJSON(w, http.StatusNoContent, nil)
}

// GetExamIntegritySummary handles GET /api/v1/exams/{id}/integrity --
// per-event-type counts across every attempt on this exam, for the
// teacher-facing quality view. Never per-student, never framed as proof
// of misconduct -- see Service.GetExamIntegritySummary's doc comment.
func (h *Handler) GetExamIntegritySummary(w http.ResponseWriter, r *http.Request) {
	examID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		middleware.WriteError(w, apperrors.BadRequest("invalid exam id"))
		return
	}
	userID, err := uuid.Parse(middleware.UserID(r.Context()))
	if err != nil {
		middleware.WriteError(w, apperrors.Unauthorized("user not authenticated"))
		return
	}
	summary, err := h.svc.GetExamIntegritySummary(r.Context(), userID, examID)
	if err != nil {
		middleware.WriteError(w, err)
		return
	}
	middleware.WriteJSON(w, http.StatusOK, summary)
}
