package handler

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/edugraph-ai/edugraph/internal/curriculum/dto"
	apperrors "github.com/edugraph-ai/edugraph/pkg/errors"
	"github.com/edugraph-ai/edugraph/pkg/middleware"
	"github.com/edugraph-ai/edugraph/pkg/validator"
)

// AddItemSkillMapping handles POST /api/v1/assessment/questions/:id/skill-mappings
// -- records a versioned Q-matrix entry (EG-GCKT spec section 6.3) and
// mirrors it to Neo4j. Mirrors AddTopicPrerequisite's handler shape.
func (h *Handler) AddItemSkillMapping(w http.ResponseWriter, r *http.Request) {
	questionID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		middleware.WriteError(w, apperrors.BadRequest("invalid question id"))
		return
	}

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

	var req dto.AddItemSkillMappingRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		middleware.WriteError(w, apperrors.BadRequest("invalid JSON body"))
		return
	}
	if err := validator.Struct(req); err != nil {
		middleware.WriteError(w, apperrors.BadRequest(err.Error()))
		return
	}

	resp, err := h.service.AddItemSkillMapping(r.Context(), userID, questionID, req)
	if err != nil {
		writeServiceError(w, err)
		return
	}

	middleware.WriteJSON(w, http.StatusCreated, resp)
}

// ListItemSkillMappings handles GET /api/v1/assessment/questions/:id/skill-mappings.
func (h *Handler) ListItemSkillMappings(w http.ResponseWriter, r *http.Request) {
	questionID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		middleware.WriteError(w, apperrors.BadRequest("invalid question id"))
		return
	}

	mappings, err := h.service.ListItemSkillMappings(r.Context(), questionID)
	if err != nil {
		writeServiceError(w, err)
		return
	}

	middleware.WriteJSON(w, http.StatusOK, mappings)
}

// ResyncItemSkillMappings handles POST /api/v1/assessment/skill-mappings/resync
// -- ministry-admin-only bulk catch-up sync, mirroring ResyncPrerequisites.
func (h *Handler) ResyncItemSkillMappings(w http.ResponseWriter, r *http.Request) {
	resp, err := h.service.ResyncItemSkillMappingsToNeo4j(r.Context())
	if err != nil {
		writeServiceError(w, err)
		return
	}

	middleware.WriteJSON(w, http.StatusOK, resp)
}
