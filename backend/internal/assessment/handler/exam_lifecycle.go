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

// CloseExam handles POST /api/v1/exams/:id/close -- moves a published exam
// to 'closed' so no further submissions are accepted (teacher/school_admin).
func (h *Handler) CloseExam(w http.ResponseWriter, r *http.Request) {
	examID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		middleware.WriteError(w, apperrors.BadRequest("invalid exam id"))
		return
	}
	callerID, err := callerID(r)
	if err != nil {
		middleware.WriteError(w, apperrors.Unauthorized("unauthenticated"))
		return
	}
	resp, err := h.svc.CloseExam(r.Context(), callerID, examID)
	if err != nil {
		middleware.WriteError(w, err)
		return
	}
	middleware.WriteJSON(w, http.StatusOK, resp)
}

// AutosaveExamDraft handles POST /api/v1/exams/:id/autosave -- upserts a
// student's in-progress answers so they can reconnect after a disconnect.
func (h *Handler) AutosaveExamDraft(w http.ResponseWriter, r *http.Request) {
	examID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		middleware.WriteError(w, apperrors.BadRequest("invalid exam id"))
		return
	}
	callerID, err := callerID(r)
	if err != nil {
		middleware.WriteError(w, apperrors.Unauthorized("unauthenticated"))
		return
	}
	var req dto.AutosaveRequest
	if !decode(w, r, &req) {
		return
	}
	if err := validator.Struct(req); err != nil {
		middleware.WriteError(w, apperrors.BadRequest(err.Error()))
		return
	}
	if err := h.svc.AutosaveExamDraft(r.Context(), callerID, examID, req); err != nil {
		middleware.WriteError(w, err)
		return
	}
	middleware.WriteJSON(w, http.StatusNoContent, nil)
}

// GetExamDraft handles GET /api/v1/exams/:id/draft -- returns a student's
// last autosaved answers for reconnect recovery.
func (h *Handler) GetExamDraft(w http.ResponseWriter, r *http.Request) {
	examID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		middleware.WriteError(w, apperrors.BadRequest("invalid exam id"))
		return
	}
	callerID, err := callerID(r)
	if err != nil {
		middleware.WriteError(w, apperrors.Unauthorized("unauthenticated"))
		return
	}
	drafts, err := h.svc.GetExamDraft(r.Context(), callerID, examID)
	if err != nil {
		middleware.WriteError(w, err)
		return
	}
	middleware.WriteJSON(w, http.StatusOK, drafts)
}
