package handler

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	apperrors "github.com/edugraph-ai/edugraph/pkg/errors"
	"github.com/edugraph-ai/edugraph/pkg/middleware"
)

// GetExamQuality handles GET /api/v1/exams/{id}/quality (Capability 4B).
// Recomputes the report from current answer data on every call -- the
// persisted row is a byproduct for 4C, not a cache the client can go
// stale on.
func (h *Handler) GetExamQuality(w http.ResponseWriter, r *http.Request) {
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
	examID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		middleware.WriteError(w, apperrors.BadRequest("invalid exam id"))
		return
	}

	resp, err := h.svc.GetExamQuality(r.Context(), userID, examID)
	if err != nil {
		middleware.WriteError(w, err)
		return
	}
	middleware.WriteJSON(w, http.StatusOK, resp)
}
