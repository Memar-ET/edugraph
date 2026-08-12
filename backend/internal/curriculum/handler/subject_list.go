package handler

import (
	"net/http"

	"github.com/edugraph-ai/edugraph/pkg/middleware"
)

// ListSubjects handles GET /api/v1/curriculum/subjects -- the Ministry
// curriculum browser: every promoted subject system-wide, not just one
// officer's own uploads (contrast with ListJobs).
func (h *Handler) ListSubjects(w http.ResponseWriter, r *http.Request) {
	items, err := h.service.ListSubjects(r.Context())
	if err != nil {
		writeServiceError(w, err)
		return
	}

	middleware.WriteJSON(w, http.StatusOK, items)
}
