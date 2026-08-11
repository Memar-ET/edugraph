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

// UpdateExamScope handles PATCH /api/v1/exams/:id/scope -- lets a teacher
// fix a wrong subject/grade/exam-type/unit-range that Capability 2A's
// title parser guessed incorrectly, without re-uploading the exam file.
// Call POST /api/v1/exams/:id/validate again afterwards to refresh the
// compliance report against the corrected scope.
func (h *Handler) UpdateExamScope(w http.ResponseWriter, r *http.Request) {
	examID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		middleware.WriteError(w, apperrors.BadRequest("invalid exam id"))
		return
	}

	var req dto.UpdateExamScopeRequest
	if !decode(w, r, &req) {
		return
	}
	if err := validator.Struct(req); err != nil {
		middleware.WriteError(w, apperrors.BadRequest(err.Error()))
		return
	}

	resp, err := h.svc.UpdateExamScope(r.Context(), examID, req)
	if err != nil {
		middleware.WriteError(w, err)
		return
	}

	middleware.WriteJSON(w, http.StatusOK, resp)
}
