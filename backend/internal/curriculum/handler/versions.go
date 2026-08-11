package handler

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/edugraph-ai/edugraph/internal/curriculum/dto"
	apperrors "github.com/edugraph-ai/edugraph/pkg/errors"
	"github.com/edugraph-ai/edugraph/pkg/middleware"
	"github.com/edugraph-ai/edugraph/pkg/validator"
)

// Supersede handles POST /api/v1/curriculum/subjects/{code}/supersede
//
// {code} must already be an approved subject (uploaded and approved
// through the normal Step 1-4 pipeline under its own code) -- this
// endpoint only records that it replaces req.PreviousCode as the current
// version of that lineage; it never copies or mutates any curriculum row.
func (h *Handler) Supersede(w http.ResponseWriter, r *http.Request) {
	newCode := chi.URLParam(r, "code")
	if newCode == "" {
		middleware.WriteError(w, apperrors.BadRequest("invalid subject code"))
		return
	}

	var req dto.SupersedeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		middleware.WriteError(w, apperrors.BadRequest("invalid JSON body"))
		return
	}
	if err := validator.Struct(req); err != nil {
		middleware.WriteError(w, apperrors.BadRequest(err.Error()))
		return
	}

	resp, err := h.service.MarkAsRevision(r.Context(), newCode, req.PreviousCode)
	if err != nil {
		writeServiceError(w, err)
		return
	}

	middleware.WriteJSON(w, http.StatusOK, resp)
}

// ListSubjectVersions handles GET /api/v1/curriculum/subjects/{code}/versions
// -- returns the subject's revision lineage oldest-first.
func (h *Handler) ListSubjectVersions(w http.ResponseWriter, r *http.Request) {
	code := chi.URLParam(r, "code")
	if code == "" {
		middleware.WriteError(w, apperrors.BadRequest("invalid subject code"))
		return
	}

	history, err := h.service.GetVersionHistory(r.Context(), code)
	if err != nil {
		writeServiceError(w, err)
		return
	}

	middleware.WriteJSON(w, http.StatusOK, history)
}
