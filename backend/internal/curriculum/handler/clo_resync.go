package handler

import (
	"net/http"

	"github.com/edugraph-ai/edugraph/pkg/middleware"
)

// ResyncCLOs handles POST /api/v1/curriculum/clos/resync (ministry_admin only).
// Re-merges every (:Topic)-[:HAS_CLO]->(:CLO) edge into Neo4j for all rows in
// curriculum.topic_clo_mappings. Used to recover from a partial graph write or
// to backfill after a Neo4j wipe.
func (h *Handler) ResyncCLOs(w http.ResponseWriter, r *http.Request) {
	result, err := h.service.ResyncCLOs(r.Context())
	if err != nil {
		writeServiceError(w, err)
		return
	}
	middleware.WriteJSON(w, http.StatusOK, result)
}
