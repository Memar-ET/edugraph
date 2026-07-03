package handler

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/edugraph-ai/edugraph/internal/assessment/dto"
	"github.com/edugraph-ai/edugraph/internal/assessment/service"
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
	var req dto.CreateAssessmentRequest
	if !decode(w, r, &req) {
		return
	}
	if err := validator.Struct(req); err != nil {
		middleware.WriteError(w, apperrors.BadRequest(err.Error()))
		return
	}
	resp, err := h.svc.CreateAssessment(r.Context(), middleware.UserID(r.Context()), req)
	if err != nil {
		middleware.WriteError(w, err)
		return
	}
	middleware.WriteJSON(w, http.StatusCreated, resp)
}

func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	resp, err := h.svc.GetAssessment(r.Context(), chi.URLParam(r, "id"))
	if err != nil {
		middleware.WriteError(w, err)
		return
	}
	middleware.WriteJSON(w, http.StatusOK, resp)
}

func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	p := pagination.FromRequest(r)
	items, total, err := h.svc.ListAssessments(r.Context(), r.URL.Query().Get("subject_id"), r.URL.Query().Get("school_id"), p)
	if err != nil {
		middleware.WriteError(w, err)
		return
	}
	middleware.WriteJSONMeta(w, http.StatusOK, items, pagination.NewMeta(p, total))
}

func (h *Handler) AddQuestion(w http.ResponseWriter, r *http.Request) {
	var req dto.CreateQuestionRequest
	if !decode(w, r, &req) {
		return
	}
	if err := validator.Struct(req); err != nil {
		middleware.WriteError(w, apperrors.BadRequest(err.Error()))
		return
	}
	resp, err := h.svc.AddQuestion(r.Context(), chi.URLParam(r, "id"), req)
	if err != nil {
		middleware.WriteError(w, err)
		return
	}
	middleware.WriteJSON(w, http.StatusCreated, resp)
}

func (h *Handler) ListQuestions(w http.ResponseWriter, r *http.Request) {
	resp, err := h.svc.ListQuestions(r.Context(), chi.URLParam(r, "id"))
	if err != nil {
		middleware.WriteError(w, err)
		return
	}
	middleware.WriteJSON(w, http.StatusOK, resp)
}

func (h *Handler) Submit(w http.ResponseWriter, r *http.Request) {
	var req dto.SubmitResultRequest
	if !decode(w, r, &req) {
		return
	}
	if err := validator.Struct(req); err != nil {
		middleware.WriteError(w, apperrors.BadRequest(err.Error()))
		return
	}
	resp, err := h.svc.SubmitResult(r.Context(), chi.URLParam(r, "id"), req)
	if err != nil {
		middleware.WriteError(w, err)
		return
	}
	middleware.WriteJSON(w, http.StatusOK, resp)
}

func (h *Handler) ResultsByAssessment(w http.ResponseWriter, r *http.Request) {
	resp, err := h.svc.ListResultsByAssessment(r.Context(), chi.URLParam(r, "id"))
	if err != nil {
		middleware.WriteError(w, err)
		return
	}
	middleware.WriteJSON(w, http.StatusOK, resp)
}

func (h *Handler) ResultsByStudent(w http.ResponseWriter, r *http.Request) {
	resp, err := h.svc.ListResultsByStudent(r.Context(), chi.URLParam(r, "studentID"))
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
