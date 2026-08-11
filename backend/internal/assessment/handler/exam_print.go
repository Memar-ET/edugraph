package handler

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	apperrors "github.com/edugraph-ai/edugraph/pkg/errors"
	"github.com/edugraph-ai/edugraph/pkg/middleware"
)

// PrintExam handles GET /api/v1/exams/:id/print (Capability 2.2, PRD
// "Mode B: Print & Paper Exam") -- returns a print-ready HTML exam sheet.
// The response is text/html, not the usual JSON envelope: it's meant to
// be opened directly in a browser tab and printed/saved as PDF from
// there, not consumed by the SPA's data layer.
func (h *Handler) PrintExam(w http.ResponseWriter, r *http.Request) {
	examID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		middleware.WriteError(w, apperrors.BadRequest("invalid exam id"))
		return
	}

	html, err := h.svc.GeneratePrintableExam(r.Context(), examID)
	if err != nil {
		middleware.WriteError(w, err)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(html))
}

// PrintAnswerKey handles GET /api/v1/exams/:id/print/answer-key --
// companion "optical answer key" reference sheet for teachers, same
// Capability 2.2. Gated to teacher/school_admin only (see router.go) --
// unlike PrintExam, this one contains the correct answers and must not be
// reachable by students.
func (h *Handler) PrintAnswerKey(w http.ResponseWriter, r *http.Request) {
	examID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		middleware.WriteError(w, apperrors.BadRequest("invalid exam id"))
		return
	}

	html, err := h.svc.GenerateAnswerKeySheet(r.Context(), examID)
	if err != nil {
		middleware.WriteError(w, err)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(html))
}
