package handler

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/edugraph-ai/edugraph/internal/reports/dto"
	"github.com/edugraph-ai/edugraph/internal/reports/service"
	apperrors "github.com/edugraph-ai/edugraph/pkg/errors"
	"github.com/edugraph-ai/edugraph/pkg/middleware"
	"github.com/edugraph-ai/edugraph/pkg/validator"
)

type Handler struct {
	svc *service.Service
}

func New(svc *service.Service) *Handler {
	return &Handler{svc: svc}
}

func (h *Handler) Generate(w http.ResponseWriter, r *http.Request) {
	var req dto.GenerateReportRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		middleware.WriteError(w, apperrors.BadRequest("invalid JSON: "+err.Error()))
		return
	}
	if err := validator.Struct(req); err != nil {
		middleware.WriteError(w, apperrors.BadRequest(err.Error()))
		return
	}
	resp, err := h.svc.Generate(r.Context(), middleware.UserID(r.Context()), req)
	if err != nil {
		middleware.WriteError(w, err)
		return
	}
	middleware.WriteJSON(w, http.StatusAccepted, resp)
}

func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	resp, err := h.svc.Get(r.Context(), middleware.UserID(r.Context()), chi.URLParam(r, "id"))
	if err != nil {
		middleware.WriteError(w, err)
		return
	}
	middleware.WriteJSON(w, http.StatusOK, resp)
}
