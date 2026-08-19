package handler

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	apperrors "github.com/edugraph-ai/edugraph/pkg/errors"
	"github.com/edugraph-ai/edugraph/pkg/middleware"
)

// ListSkillStateSnapshots handles
// GET /api/v1/students/:id/topics/:topicId/state-snapshots
// (EG-GCKT checklist sections 6/18/22) -- the historical record of a
// student's fused knowledge state for one topic, for comparison over
// time. studentID is caller-supplied; ownership is enforced server-side
// the same way as Explain.
func (h *Handler) ListSkillStateSnapshots(w http.ResponseWriter, r *http.Request) {
	studentID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		middleware.WriteError(w, apperrors.BadRequest("invalid student id"))
		return
	}
	topicID, err := uuid.Parse(chi.URLParam(r, "topicId"))
	if err != nil {
		middleware.WriteError(w, apperrors.BadRequest("invalid topic id"))
		return
	}
	callerID, err := uuid.Parse(middleware.UserID(r.Context()))
	if err != nil {
		middleware.WriteError(w, apperrors.Unauthorized("user not authenticated"))
		return
	}

	snapshots, err := h.svc.ListSkillStateSnapshots(r.Context(), callerID, studentID, topicID)
	if err != nil {
		middleware.WriteError(w, err)
		return
	}

	middleware.WriteJSON(w, http.StatusOK, snapshots)
}
