package handler

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/edugraph-ai/edugraph/internal/notification/dto"
	"github.com/edugraph-ai/edugraph/internal/notification/service"
	apperrors "github.com/edugraph-ai/edugraph/pkg/errors"
	"github.com/edugraph-ai/edugraph/pkg/middleware"
	"github.com/edugraph-ai/edugraph/pkg/pagination"
	"github.com/edugraph-ai/edugraph/pkg/validator"
)

type Handler struct {
	svc *service.Service
}

func New(svc *service.Service) *Handler {
	return &Handler{svc: svc}
}

func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	var req dto.CreateNotificationRequest
	defer r.Body.Close()
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		middleware.WriteError(w, apperrors.BadRequest("invalid request body"))
		return
	}
	if err := validator.Struct(req); err != nil {
		middleware.WriteError(w, apperrors.BadRequest(err.Error()))
		return
	}
	resp, err := h.svc.Create(r.Context(), req)
	if err != nil {
		middleware.WriteError(w, err)
		return
	}
	middleware.WriteJSON(w, http.StatusCreated, resp)
}

func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	userID := middleware.UserID(r.Context())
	unreadOnly := r.URL.Query().Get("unread") == "true"
	p := pagination.FromRequest(r)

	items, total, err := h.svc.ListForUser(r.Context(), userID, unreadOnly, p)
	if err != nil {
		middleware.WriteError(w, err)
		return
	}
	middleware.WriteJSONMeta(w, http.StatusOK, items, pagination.NewMeta(p, total))
}

func (h *Handler) MarkRead(w http.ResponseWriter, r *http.Request) {
	userID := middleware.UserID(r.Context())
	if err := h.svc.MarkRead(r.Context(), chi.URLParam(r, "id"), userID); err != nil {
		middleware.WriteError(w, err)
		return
	}
	middleware.WriteJSON(w, http.StatusNoContent, nil)
}
