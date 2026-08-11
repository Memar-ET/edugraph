package handler

import (
	"net/http"
	"strconv"

	"github.com/google/uuid"

	apperrors "github.com/edugraph-ai/edugraph/pkg/errors"
	"github.com/edugraph-ai/edugraph/pkg/middleware"
)

// GetClassHeatmap handles GET /api/v1/teachers/me/class-heatmap
// ?subjectCode=BIO&gradeLevel=11 (Capability 4A). The class is scoped to
// the caller's own school (resolved server-side from their user row --
// never a client-supplied school id), teacher or school_admin role.
func (h *Handler) GetClassHeatmap(w http.ResponseWriter, r *http.Request) {
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

	subjectCode := r.URL.Query().Get("subjectCode")
	if subjectCode == "" {
		middleware.WriteError(w, apperrors.BadRequest("subjectCode query parameter is required"))
		return
	}
	gradeLevel, err := strconv.Atoi(r.URL.Query().Get("gradeLevel"))
	if err != nil || gradeLevel < 1 || gradeLevel > 12 {
		middleware.WriteError(w, apperrors.BadRequest("gradeLevel must be a number between 1 and 12"))
		return
	}

	resp, err := h.svc.GetClassHeatmap(r.Context(), userID, subjectCode, gradeLevel)
	if err != nil {
		middleware.WriteError(w, err)
		return
	}

	middleware.WriteJSON(w, http.StatusOK, resp)
}
