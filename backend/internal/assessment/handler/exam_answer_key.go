package handler

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	apperrors "github.com/edugraph-ai/edugraph/pkg/errors"
	"github.com/edugraph-ai/edugraph/pkg/fileformat"
	"github.com/edugraph-ai/edugraph/pkg/middleware"
)

const maxAnswerKeyUploadBytes = 10 << 20 // 10 MB -- a small table document, not a full exam

// UploadAnswerKey handles POST /api/v1/exams/:id/answer-key -- a separate
// PDF/DOCX upload carrying just the Q#/Correct-Answer table, applied async
// to the exam's already-parsed questions.
func (h *Handler) UploadAnswerKey(w http.ResponseWriter, r *http.Request) {
	examID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		middleware.WriteError(w, apperrors.BadRequest("invalid exam id"))
		return
	}

	if err := r.ParseMultipartForm(maxAnswerKeyUploadBytes); err != nil {
		middleware.WriteError(w, apperrors.BadRequest("invalid multipart form data"))
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		middleware.WriteError(w, apperrors.BadRequest("file is required"))
		return
	}
	defer file.Close()

	mimeType, ok := fileformat.SniffPDFOrDOCX(file)
	if !ok {
		middleware.WriteError(w, apperrors.BadRequest("file content is not a valid PDF or DOCX document"))
		return
	}

	resp, err := h.svc.UploadAnswerKey(r.Context(), examID, header.Filename, mimeType, file)
	if err != nil {
		middleware.WriteError(w, err)
		return
	}

	middleware.WriteJSON(w, http.StatusAccepted, resp)
}
