package handler

import (
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"github.com/edugraph-ai/edugraph/pkg/middleware"
)

// UnderperformingSchools handles GET /api/v1/ministry/regions/{regionID}/underperforming
func (h *Handler) UnderperformingSchools(w http.ResponseWriter, r *http.Request) {
	regionID := chi.URLParam(r, "regionID")
	limit := 10
	if l := r.URL.Query().Get("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil && parsed > 0 {
			limit = parsed
		}
	}

	resp, err := h.svc.UnderperformingSchools(r.Context(), regionID, limit)
	if err != nil {
		middleware.WriteError(w, err)
		return
	}
	middleware.WriteJSON(w, http.StatusOK, resp)
}
