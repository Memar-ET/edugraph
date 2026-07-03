package handler

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"github.com/edugraph-ai/edugraph/internal/curriculum/dto"
	"github.com/edugraph-ai/edugraph/internal/curriculum/service"
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

func (h *Handler) CreateSubject(w http.ResponseWriter, r *http.Request) {
	var req dto.CreateSubjectRequest
	if !decode(w, r, &req) {
		return
	}
	if err := validator.Struct(req); err != nil {
		middleware.WriteError(w, apperrors.BadRequest(err.Error()))
		return
	}
	resp, err := h.svc.CreateSubject(r.Context(), req)
	if err != nil {
		middleware.WriteError(w, err)
		return
	}
	middleware.WriteJSON(w, http.StatusCreated, resp)
}

func (h *Handler) ListSubjects(w http.ResponseWriter, r *http.Request) {
	grade, _ := strconv.Atoi(r.URL.Query().Get("grade_level"))
	resp, err := h.svc.ListSubjects(r.Context(), int16(grade))
	if err != nil {
		middleware.WriteError(w, err)
		return
	}
	middleware.WriteJSON(w, http.StatusOK, resp)
}

func (h *Handler) CreateUnit(w http.ResponseWriter, r *http.Request) {
	var req dto.CreateUnitRequest
	if !decode(w, r, &req) {
		return
	}
	if err := validator.Struct(req); err != nil {
		middleware.WriteError(w, apperrors.BadRequest(err.Error()))
		return
	}
	resp, err := h.svc.CreateUnit(r.Context(), req)
	if err != nil {
		middleware.WriteError(w, err)
		return
	}
	middleware.WriteJSON(w, http.StatusCreated, resp)
}

func (h *Handler) GetUnit(w http.ResponseWriter, r *http.Request) {
	resp, err := h.svc.GetUnit(r.Context(), chi.URLParam(r, "id"))
	if err != nil {
		middleware.WriteError(w, err)
		return
	}
	middleware.WriteJSON(w, http.StatusOK, resp)
}

func (h *Handler) ListUnits(w http.ResponseWriter, r *http.Request) {
	resp, err := h.svc.ListUnits(r.Context(), r.URL.Query().Get("subject_id"))
	if err != nil {
		middleware.WriteError(w, err)
		return
	}
	middleware.WriteJSON(w, http.StatusOK, resp)
}

func (h *Handler) UpdateUnit(w http.ResponseWriter, r *http.Request) {
	var req dto.UpdateUnitRequest
	if !decode(w, r, &req) {
		return
	}
	if err := validator.Struct(req); err != nil {
		middleware.WriteError(w, apperrors.BadRequest(err.Error()))
		return
	}
	resp, err := h.svc.UpdateUnit(r.Context(), chi.URLParam(r, "id"), req)
	if err != nil {
		middleware.WriteError(w, err)
		return
	}
	middleware.WriteJSON(w, http.StatusOK, resp)
}

func (h *Handler) DeleteUnit(w http.ResponseWriter, r *http.Request) {
	if err := h.svc.DeleteUnit(r.Context(), chi.URLParam(r, "id")); err != nil {
		middleware.WriteError(w, err)
		return
	}
	middleware.WriteJSON(w, http.StatusNoContent, nil)
}

func (h *Handler) AddPrerequisite(w http.ResponseWriter, r *http.Request) {
	var req dto.AddPrerequisiteRequest
	if !decode(w, r, &req) {
		return
	}
	if err := validator.Struct(req); err != nil {
		middleware.WriteError(w, apperrors.BadRequest(err.Error()))
		return
	}
	if err := h.svc.AddPrerequisite(r.Context(), chi.URLParam(r, "id"), req); err != nil {
		middleware.WriteError(w, err)
		return
	}
	middleware.WriteJSON(w, http.StatusNoContent, nil)
}

func (h *Handler) Prerequisites(w http.ResponseWriter, r *http.Request) {
	resp, err := h.svc.Prerequisites(r.Context(), chi.URLParam(r, "id"))
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
