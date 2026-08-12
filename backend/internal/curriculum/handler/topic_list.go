package handler

import (
	"net/http"

	"github.com/go-chi/chi/v5"

	apperrors "github.com/edugraph-ai/edugraph/pkg/errors"
	"github.com/edugraph-ai/edugraph/pkg/middleware"
)

// ListTopicsBySubject handles GET /api/v1/curriculum/subjects/{code}/topics
// -- backs the prerequisites UI's topic picker.
func (h *Handler) ListTopicsBySubject(w http.ResponseWriter, r *http.Request) {
	code := chi.URLParam(r, "code")
	if code == "" {
		middleware.WriteError(w, apperrors.BadRequest("invalid subject code"))
		return
	}

	items, err := h.service.ListTopicsBySubject(r.Context(), code)
	if err != nil {
		writeServiceError(w, err)
		return
	}

	middleware.WriteJSON(w, http.StatusOK, items)
}
