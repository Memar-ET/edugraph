package handler

import (
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/edugraph-ai/edugraph/pkg/middleware"
)

// QMatrixQuality handles GET /api/v1/curriculum/subjects/:code/qmatrix-quality
// -- EG-GCKT checklist: "detect items with missing or low-confidence
// skill mappings" / "flag weak or ambiguous Q-matrix mappings."
func (h *Handler) QMatrixQuality(w http.ResponseWriter, r *http.Request) {
	subjectCode := chi.URLParam(r, "code")

	report, err := h.service.QMatrixQuality(r.Context(), subjectCode)
	if err != nil {
		writeServiceError(w, err)
		return
	}

	middleware.WriteJSON(w, http.StatusOK, report)
}

// PrerequisiteQuality handles GET /api/v1/curriculum/subjects/:code/prerequisite-quality
// -- EG-GCKT checklist: "run structural validation for orphaned skills...
// and duplicate edges."
func (h *Handler) PrerequisiteQuality(w http.ResponseWriter, r *http.Request) {
	subjectCode := chi.URLParam(r, "code")

	report, err := h.service.PrerequisiteQuality(r.Context(), subjectCode)
	if err != nil {
		writeServiceError(w, err)
		return
	}

	middleware.WriteJSON(w, http.StatusOK, report)
}
