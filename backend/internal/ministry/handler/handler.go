package handler

import (
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/edugraph-ai/edugraph/internal/ministry/service"
	"github.com/edugraph-ai/edugraph/pkg/middleware"
)

type Handler struct {
	svc *service.Service
}

func New(svc *service.Service) *Handler {
	return &Handler{svc: svc}
}

func (h *Handler) Overview(w http.ResponseWriter, r *http.Request) {
	resp, err := h.svc.Overview(r.Context())
	if err != nil {
		middleware.WriteError(w, err)
		return
	}
	middleware.WriteJSON(w, http.StatusOK, resp)
}

func (h *Handler) RegionStats(w http.ResponseWriter, r *http.Request) {
	resp, err := h.svc.RegionStats(r.Context(), chi.URLParam(r, "regionID"))
	if err != nil {
		middleware.WriteError(w, err)
		return
	}
	middleware.WriteJSON(w, http.StatusOK, resp)
}
