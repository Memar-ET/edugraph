package handler

import (
	"encoding/json"
	"net/http"

	"github.com/edugraph-ai/edugraph/internal/storage/dto"
	"github.com/edugraph-ai/edugraph/internal/storage/service"
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

func (h *Handler) PresignUpload(w http.ResponseWriter, r *http.Request) {
	var req dto.PresignUploadRequest
	if !decode(w, r, &req) {
		return
	}
	if err := validator.Struct(req); err != nil {
		middleware.WriteError(w, apperrors.BadRequest(err.Error()))
		return
	}
	resp, err := h.svc.PresignUpload(r.Context(), req)
	if err != nil {
		middleware.WriteError(w, err)
		return
	}
	middleware.WriteJSON(w, http.StatusOK, resp)
}

func (h *Handler) PresignDownload(w http.ResponseWriter, r *http.Request) {
	var req dto.PresignDownloadRequest
	if !decode(w, r, &req) {
		return
	}
	if err := validator.Struct(req); err != nil {
		middleware.WriteError(w, apperrors.BadRequest(err.Error()))
		return
	}
	resp, err := h.svc.PresignDownload(r.Context(), req)
	if err != nil {
		middleware.WriteError(w, err)
		return
	}
	middleware.WriteJSON(w, http.StatusOK, resp)
}

func decode(w http.ResponseWriter, r *http.Request, dst any) bool {
	defer r.Body.Close()
	if err := json.NewDecoder(r.Body).Decode(dst); err != nil {
		middleware.WriteError(w, apperrors.BadRequest("invalid request body"))
		return false
	}
	return true
}
