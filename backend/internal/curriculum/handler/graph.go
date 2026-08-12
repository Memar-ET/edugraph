package handler

import (
	"net/http"

	"github.com/go-chi/chi/v5"

	apperrors "github.com/edugraph-ai/edugraph/pkg/errors"
	"github.com/edugraph-ai/edugraph/pkg/middleware"
)

// GetSubjectGraph handles GET /api/v1/curriculum/subjects/{code}/graph
// -- backs the frontend's Neo4j knowledge-graph visualization.
// ?includeClos=true additionally includes CLO nodes (excluded by default
// since a subject's CLO count can be several times its topic count).
func (h *Handler) GetSubjectGraph(w http.ResponseWriter, r *http.Request) {
	code := chi.URLParam(r, "code")
	if code == "" {
		middleware.WriteError(w, apperrors.BadRequest("invalid subject code"))
		return
	}
	includeClos := r.URL.Query().Get("includeClos") == "true"

	graph, err := h.service.GetSubjectGraph(r.Context(), code, includeClos)
	if err != nil {
		writeServiceError(w, err)
		return
	}

	middleware.WriteJSON(w, http.StatusOK, graph)
}
