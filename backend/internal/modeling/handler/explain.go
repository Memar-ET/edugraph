package handler

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	apperrors "github.com/edugraph-ai/edugraph/pkg/errors"
	"github.com/edugraph-ai/edugraph/pkg/middleware"
)

// Explain handles GET /api/v1/students/:id/topics/:topicId/explain
// (EG-GCKT Milestone 11, spec section 18) -- the five-part explanation
// (current state, evidence, structural context, confidence,
// recommendation + reason) for one student's estimated knowledge of one
// topic. studentID is caller-supplied, so the service layer enforces
// server-side ownership (own record for a student, same-school for a
// teacher/school_admin) -- see Service.authorizeExplain.
func (h *Handler) Explain(w http.ResponseWriter, r *http.Request) {
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

	explanation, err := h.svc.Explain(r.Context(), callerID, studentID, topicID)
	if err != nil {
		middleware.WriteError(w, err)
		return
	}

	middleware.WriteJSON(w, http.StatusOK, explanation)
}
