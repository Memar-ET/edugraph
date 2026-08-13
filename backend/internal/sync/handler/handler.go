package handler

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/edugraph-ai/edugraph/internal/sync/dto"
	"github.com/edugraph-ai/edugraph/internal/sync/service"
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

func (h *Handler) Push(w http.ResponseWriter, r *http.Request) {
	var req dto.PushRequest
	defer r.Body.Close()
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		middleware.WriteError(w, apperrors.BadRequest("invalid request body"))
		return
	}
	if err := validator.Struct(req); err != nil {
		middleware.WriteError(w, apperrors.BadRequest(err.Error()))
		return
	}

	resp, err := h.svc.Push(r.Context(), deviceSchoolID(r.Context()), req)
	if err != nil {
		middleware.WriteError(w, err)
		return
	}
	middleware.WriteJSON(w, http.StatusAccepted, resp)
}

func (h *Handler) Pull(w http.ResponseWriter, r *http.Request) {
	schoolID := r.URL.Query().Get("school_id")
	if schoolID == "" {
		middleware.WriteError(w, apperrors.BadRequest("school_id is required"))
		return
	}

	since := time.Unix(0, 0)
	if raw := r.URL.Query().Get("since"); raw != "" {
		parsed, err := time.Parse(time.RFC3339, raw)
		if err != nil {
			middleware.WriteError(w, apperrors.BadRequest("since must be RFC3339"))
			return
		}
		since = parsed
	}

	resp, err := h.svc.Pull(r.Context(), deviceSchoolID(r.Context()), schoolID, since)
	if err != nil {
		middleware.WriteError(w, err)
		return
	}
	middleware.WriteJSON(w, http.StatusOK, resp)
}
