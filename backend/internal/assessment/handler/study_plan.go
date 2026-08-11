package handler

import (
	"net/http"

	"github.com/google/uuid"

	"github.com/edugraph-ai/edugraph/internal/assessment/dto"
	apperrors "github.com/edugraph-ai/edugraph/pkg/errors"
	"github.com/edugraph-ai/edugraph/pkg/middleware"
	"github.com/edugraph-ai/edugraph/pkg/validator"
)

// GenerateStudyPlan handles POST /api/v1/students/me/study-plans (role
// student) -- queues Capability 3B generation; the plan appears on the
// GET shortly after.
func (h *Handler) GenerateStudyPlan(w http.ResponseWriter, r *http.Request) {
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

	var req dto.GenerateStudyPlanRequest
	if r.ContentLength != 0 && !decode(w, r, &req) {
		return
	}
	if err := validator.Struct(req); err != nil {
		middleware.WriteError(w, apperrors.BadRequest(err.Error()))
		return
	}

	resp, err := h.svc.GenerateStudyPlan(r.Context(), userID, req)
	if err != nil {
		middleware.WriteError(w, err)
		return
	}

	middleware.WriteJSON(w, http.StatusAccepted, resp)
}

// ListMyStudyPlans handles GET /api/v1/students/me/study-plans (role
// student) -- the active day-by-day plans generated from gap records.
func (h *Handler) ListMyStudyPlans(w http.ResponseWriter, r *http.Request) {
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

	plans, err := h.svc.ListMyStudyPlans(r.Context(), userID)
	if err != nil {
		middleware.WriteError(w, err)
		return
	}

	middleware.WriteJSON(w, http.StatusOK, plans)
}
