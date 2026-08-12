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

// AddTopicPrerequisite handles POST /api/v1/curriculum/topics/:id/prerequisites
// -- records "this topic requires that one first" and mirrors the edge to
// Neo4j, which is what makes gap-analysis root causes (3A) and study-plan
// ordering (3B) come alive.
func (h *Handler) AddTopicPrerequisite(w http.ResponseWriter, r *http.Request) {
	topicID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		middleware.WriteError(w, apperrors.BadRequest("invalid topic id"))
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

	var req dto.AddPrerequisiteRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		middleware.WriteError(w, apperrors.BadRequest("invalid JSON body"))
		return
	}
	if err := validator.Struct(req); err != nil {
		middleware.WriteError(w, apperrors.BadRequest(err.Error()))
		return
	}

	resp, err := h.service.AddTopicPrerequisite(r.Context(), userID, topicID, req)
	if err != nil {
		writeServiceError(w, err)
		return
	}

	middleware.WriteJSON(w, http.StatusCreated, resp)
}

// ListTopicPrerequisites handles GET /api/v1/curriculum/topics/:id/prerequisites.
func (h *Handler) ListTopicPrerequisites(w http.ResponseWriter, r *http.Request) {
	topicID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		middleware.WriteError(w, apperrors.BadRequest("invalid topic id"))
		return
	}

	links, err := h.service.ListTopicPrerequisites(r.Context(), topicID)
	if err != nil {
		writeServiceError(w, err)
		return
	}

	middleware.WriteJSON(w, http.StatusOK, links)
}

// ValidatePrerequisite handles
// PATCH /api/v1/curriculum/topics/:id/prerequisites/:prereqId/validate
//
// Confirms an existing link (typically one created with inferMethod
// "ai_inferred", so IsValidated started false) -- feature 1.4.
func (h *Handler) ValidatePrerequisite(w http.ResponseWriter, r *http.Request) {
	topicID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		middleware.WriteError(w, apperrors.BadRequest("invalid topic id"))
		return
	}
	prereqID, err := uuid.Parse(chi.URLParam(r, "prereqId"))
	if err != nil {
		middleware.WriteError(w, apperrors.BadRequest("invalid prerequisite topic id"))
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

	resp, err := h.service.ValidatePrerequisite(r.Context(), userID, topicID, prereqID)
	if err != nil {
		writeServiceError(w, err)
		return
	}

	middleware.WriteJSON(w, http.StatusOK, resp)
}

// ResyncPrerequisites handles POST /api/v1/curriculum/prerequisites/resync
//
// Re-mirrors every prerequisite link Postgres doesn't know is current in
// Neo4j (neo4j_written = false) -- feature 1.5. Ministry-admin only: it
// walks the whole graph, not one topic, so it's an operational/ops action
// rather than part of the normal curriculum-editing flow.
func (h *Handler) ResyncPrerequisites(w http.ResponseWriter, r *http.Request) {
	resp, err := h.service.ResyncPrerequisitesToNeo4j(r.Context())
	if err != nil {
		writeServiceError(w, err)
		return
	}

	middleware.WriteJSON(w, http.StatusOK, resp)
}
