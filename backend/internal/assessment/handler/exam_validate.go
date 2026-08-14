package handler

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	apperrors "github.com/edugraph-ai/edugraph/pkg/errors"
	"github.com/edugraph-ai/edugraph/pkg/middleware"
)

// ValidateExam handles POST /api/v1/exams/:id/validate -- computes and
// stores Capability 2B's 5-part report, moving the exam to
// 'validation_pending'.
func (h *Handler) ValidateExam(w http.ResponseWriter, r *http.Request) {
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

	report, err := h.svc.ValidateExam(r.Context(), userID, examID)
	if err != nil {
		middleware.WriteError(w, err)
		return
	}

	middleware.WriteJSON(w, http.StatusOK, report)
}

// PublishExam handles POST /api/v1/exams/:id/publish -- only succeeds once
// the exam has been validated at least once.
func (h *Handler) PublishExam(w http.ResponseWriter, r *http.Request) {
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

	resp, err := h.svc.PublishExam(r.Context(), userID, examID)
	if err != nil {
		middleware.WriteError(w, err)
		return
	}

	middleware.WriteJSON(w, http.StatusOK, resp)
}
