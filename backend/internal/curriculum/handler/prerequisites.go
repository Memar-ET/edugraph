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
