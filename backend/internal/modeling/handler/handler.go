package handler

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/edugraph-ai/edugraph/internal/modeling/service"
	apperrors "github.com/edugraph-ai/edugraph/pkg/errors"
	"github.com/edugraph-ai/edugraph/pkg/middleware"
)

// Handler backs the EG-GCKT model-governance review queue (Milestone 9):
// reviewing candidate parameter refits (BKT/DINA/IRT, produced nightly by
// ai-service's refit_worker.py) before they can affect the live engines.
type Handler struct {
	svc *service.Service
}

func New(svc *service.Service) *Handler {
	return &Handler{svc: svc}
}

func callerID(r *http.Request) (uuid.UUID, error) {
	return uuid.Parse(middleware.UserID(r.Context()))
}

// ListCandidateSnapshots handles GET /api/v1/model-snapshots/candidates
// (role ministry_admin/curriculum_officer).
func (h *Handler) ListCandidateSnapshots(w http.ResponseWriter, r *http.Request) {
	snapshots, err := h.svc.ListCandidateSnapshots(r.Context())
	if err != nil {
		middleware.WriteError(w, err)
		return
	}
	middleware.WriteJSON(w, http.StatusOK, snapshots)
}

// PromoteSnapshot handles POST /api/v1/model-snapshots/:id/promote
// (role ministry_admin) -- activates a candidate refit.
func (h *Handler) PromoteSnapshot(w http.ResponseWriter, r *http.Request) {
	snapshotID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		middleware.WriteError(w, apperrors.BadRequest("invalid snapshot id"))
		return
	}
	userID, err := callerID(r)
	if err != nil {
		middleware.WriteError(w, apperrors.Internal(err))
		return
	}

	snapshot, err := h.svc.PromoteSnapshot(r.Context(), userID, snapshotID)
	if err != nil {
		middleware.WriteError(w, err)
		return
	}
	middleware.WriteJSON(w, http.StatusOK, snapshot)
}

// RejectSnapshot handles POST /api/v1/model-snapshots/:id/reject
// (role ministry_admin) -- declines a candidate refit.
func (h *Handler) RejectSnapshot(w http.ResponseWriter, r *http.Request) {
	snapshotID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		middleware.WriteError(w, apperrors.BadRequest("invalid snapshot id"))
		return
	}
	userID, err := callerID(r)
	if err != nil {
		middleware.WriteError(w, apperrors.Internal(err))
		return
	}

	snapshot, err := h.svc.RejectSnapshot(r.Context(), userID, snapshotID)
	if err != nil {
		middleware.WriteError(w, err)
		return
	}
	middleware.WriteJSON(w, http.StatusOK, snapshot)
}
