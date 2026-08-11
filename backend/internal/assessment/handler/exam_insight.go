package handler

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	apperrors "github.com/edugraph-ai/edugraph/pkg/errors"
	"github.com/edugraph-ai/edugraph/pkg/middleware"
)

// GetMyExamInsight handles GET /api/v1/exams/:id/my-insight (role
// student) -- the Capability 3A narrative summary + gap records for the
// authenticated student's attempt. 404 until the async analysis lands.
func (h *Handler) GetMyExamInsight(w http.ResponseWriter, r *http.Request) {
	examID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		middleware.WriteError(w, apperrors.BadRequest("invalid exam id"))
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

	insight, err := h.svc.GetMyExamInsight(r.Context(), userID, examID)
	if err != nil {
		middleware.WriteError(w, err)
		return
	}

	middleware.WriteJSON(w, http.StatusOK, insight)
}

// ListExamInsights handles GET /api/v1/exams/:id/insights (role
// teacher/school_admin) -- one summary row per analyzed attempt.
func (h *Handler) ListExamInsights(w http.ResponseWriter, r *http.Request) {
	examID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		middleware.WriteError(w, apperrors.BadRequest("invalid exam id"))
		return
	}

	insights, err := h.svc.ListExamInsights(r.Context(), examID)
	if err != nil {
		middleware.WriteError(w, err)
		return
	}

	middleware.WriteJSON(w, http.StatusOK, insights)
}

// GetMySubjectProfiles handles GET /api/v1/students/me/subject-profiles
// (role student) -- the Subject Health Layer for the student dashboard.
func (h *Handler) GetMySubjectProfiles(w http.ResponseWriter, r *http.Request) {
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

	profiles, err := h.svc.GetMySubjectProfiles(r.Context(), userID)
	if err != nil {
		middleware.WriteError(w, err)
		return
	}

	middleware.WriteJSON(w, http.StatusOK, profiles)
}
