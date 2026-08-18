package handler

import (
	"net/http"

	"github.com/google/uuid"

	apperrors "github.com/edugraph-ai/edugraph/pkg/errors"
	"github.com/edugraph-ai/edugraph/pkg/middleware"
)

// GetAvailableExams handles GET /api/v1/students/me/available-exams.
func (h *Handler) GetAvailableExams(w http.ResponseWriter, r *http.Request) {
	userID, err := uuid.Parse(middleware.UserID(r.Context()))
	if err != nil {
		middleware.WriteError(w, apperrors.Unauthorized("user not authenticated"))
		return
	}

	resp, err := h.svc.GetAvailableExams(r.Context(), userID)
	if err != nil {
		middleware.WriteError(w, err)
		return
	}

	middleware.WriteJSON(w, http.StatusOK, resp)
}
