package handler

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	apperrors "github.com/edugraph-ai/edugraph/pkg/errors"
	"github.com/edugraph-ai/edugraph/pkg/middleware"
)

// GetMySkillStates handles GET /api/v1/students/me/skill-states -- the
// authenticated student's own full skill map (every topic with any
// evidence), backing the student-facing skill-map view.
func (h *Handler) GetMySkillStates(w http.ResponseWriter, r *http.Request) {
	callerID, err := uuid.Parse(middleware.UserID(r.Context()))
	if err != nil {
		middleware.WriteError(w, apperrors.Unauthorized("user not authenticated"))
		return
	}

	resp, err := h.svc.MySkillStates(r.Context(), callerID)
	if err != nil {
		middleware.WriteError(w, err)
		return
	}

	middleware.WriteJSON(w, http.StatusOK, resp)
}

// GetStudentSkillStates handles GET /api/v1/students/{id}/skill-states --
// the teacher/school_admin-facing counterpart, used by
// TeacherStudentDetailPage. studentID is caller-supplied;
// Service.SkillStatesForStudent enforces ownership server-side.
func (h *Handler) GetStudentSkillStates(w http.ResponseWriter, r *http.Request) {
	studentID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		middleware.WriteError(w, apperrors.BadRequest("invalid student id"))
		return
	}
	callerID, err := uuid.Parse(middleware.UserID(r.Context()))
	if err != nil {
		middleware.WriteError(w, apperrors.Unauthorized("user not authenticated"))
		return
	}

	resp, err := h.svc.SkillStatesForStudent(r.Context(), callerID, studentID)
	if err != nil {
		middleware.WriteError(w, err)
		return
	}

	middleware.WriteJSON(w, http.StatusOK, resp)
}
