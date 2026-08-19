package handler

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	apperrors "github.com/edugraph-ai/edugraph/pkg/errors"
	"github.com/edugraph-ai/edugraph/pkg/middleware"
)

// StartAttempt handles POST /api/v1/exams/{id}/start -- creates (or
// resumes, if one is already in_progress) the calling student's exam
// session. This is the only place an assessment.exam_attempts row gets
// created; questions/autosave/submit all now require one to already
// exist.
func (h *Handler) StartAttempt(w http.ResponseWriter, r *http.Request) {
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

	resp, err := h.svc.StartAttempt(r.Context(), userID, examID)
	if err != nil {
		middleware.WriteError(w, err)
		return
	}

	middleware.WriteJSON(w, http.StatusOK, resp)
}
