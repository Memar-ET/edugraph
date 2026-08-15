package handler

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/edugraph-ai/edugraph/internal/assessment/dto"
	apperrors "github.com/edugraph-ai/edugraph/pkg/errors"
	"github.com/edugraph-ai/edugraph/pkg/middleware"
	"github.com/edugraph-ai/edugraph/pkg/validator"
)

// ListCandidateMisconceptions handles GET /api/v1/misconceptions
// (role teacher/school_admin) -- the EG-GCKT Milestone 6 review queue.
func (h *Handler) ListCandidateMisconceptions(w http.ResponseWriter, r *http.Request) {
	userID, err := callerID(r)
	if err != nil {
		middleware.WriteError(w, apperrors.Internal(err))
		return
	}

	hypotheses, err := h.svc.ListCandidateMisconceptions(r.Context(), userID)
	if err != nil {
		middleware.WriteError(w, err)
		return
	}

	middleware.WriteJSON(w, http.StatusOK, hypotheses)
}

// ReviewMisconception handles PATCH /api/v1/misconceptions/:id/review
// (role teacher/school_admin) -- confirm or reject a candidate hypothesis.
func (h *Handler) ReviewMisconception(w http.ResponseWriter, r *http.Request) {
	misconceptionID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		middleware.WriteError(w, apperrors.BadRequest("invalid misconception id"))
		return
	}
	userID, err := callerID(r)
	if err != nil {
		middleware.WriteError(w, apperrors.Internal(err))
		return
	}

	var req dto.ReviewMisconceptionRequest
	if !decode(w, r, &req) {
		return
	}
	if err := validator.Struct(req); err != nil {
		middleware.WriteError(w, apperrors.BadRequest(err.Error()))
		return
	}

	hypothesis, err := h.svc.ReviewMisconception(r.Context(), userID, misconceptionID, req)
	if err != nil {
		middleware.WriteError(w, err)
		return
	}

	middleware.WriteJSON(w, http.StatusOK, hypothesis)
}
