package handler

import (
	"net/http"

	"github.com/google/uuid"

	apperrors "github.com/edugraph-ai/edugraph/pkg/errors"
	"github.com/edugraph-ai/edugraph/pkg/middleware"
	"github.com/edugraph-ai/edugraph/pkg/pagination"
)

// ListJobs handles GET /api/v1/curriculum/jobs -- the Curriculum Officer
// dashboard's job history (feature 1.2), scoped to the authenticated
// user's own uploads so an officer can track status without knowing a
// job's URL/ID.
func (h *Handler) ListJobs(w http.ResponseWriter, r *http.Request) {
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

	p := pagination.FromRequest(r)
	items, total, err := h.service.ListJobs(r.Context(), userID, p)
	if err != nil {
		writeServiceError(w, err)
		return
	}

	middleware.WriteJSONMeta(w, http.StatusOK, items, pagination.NewMeta(p, total))
}
