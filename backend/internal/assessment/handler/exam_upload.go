package handler

import (
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/edugraph-ai/edugraph/internal/assessment/dto"
	apperrors "github.com/edugraph-ai/edugraph/pkg/errors"
	"github.com/edugraph-ai/edugraph/pkg/fileformat"
	"github.com/edugraph-ai/edugraph/pkg/middleware"
	"github.com/edugraph-ai/edugraph/pkg/validator"
)

const maxExamUploadBytes = 25 << 20 // 25 MB, matches curriculum's cap

// UploadExam handles POST /api/v1/exams/upload
// Expects multipart/form-data with a "file" part (PDF or DOCX) plus form
// fields "title", "academicYear", "totalMarks", and optionally "term".
// subjectCode/gradeLevel/examScope are NOT form fields -- they're derived
// server-side from the title (see service.UploadExam). Requires an
// authenticated user with the teacher or school_admin role (enforced by
// middleware.RequireRole in the router).
func (h *Handler) UploadExam(w http.ResponseWriter, r *http.Request) {
	userIDStr := middleware.UserID(r.Context())
	if userIDStr == "" {
		middleware.WriteError(w, apperrors.Unauthorized("user not authenticated"))
		return
	}
	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		middleware.WriteError(w, apperrors.Internal(err))
		return
	}

	if err := r.ParseMultipartForm(maxExamUploadBytes); err != nil {
		middleware.WriteError(w, apperrors.BadRequest("invalid multipart form data"))
		return
	}

	totalMarks, err := strconv.Atoi(r.FormValue("totalMarks"))
	if err != nil {
		middleware.WriteError(w, apperrors.BadRequest("totalMarks must be a number"))
		return
	}

	req := dto.UploadExamRequest{
		Title:        r.FormValue("title"),
		AcademicYear: r.FormValue("academicYear"),
		TotalMarks:   totalMarks,
	}
	if termStr := r.FormValue("term"); termStr != "" {
		term, err := strconv.Atoi(termStr)
		if err != nil {
			middleware.WriteError(w, apperrors.BadRequest("term must be a number between 1 and 4"))
			return
		}
		req.Term = &term
	}
	if err := validator.Struct(req); err != nil {
		middleware.WriteError(w, apperrors.BadRequest(err.Error()))
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

	resp, err := h.svc.UploadExam(r.Context(), userID, req, header.Filename, mimeType, header.Size, file)
	if err != nil {
		middleware.WriteError(w, err)
		return
	}

	middleware.WriteJSON(w, http.StatusAccepted, resp)
}

// GetExam handles GET /api/v1/exams/:id
func (h *Handler) GetExam(w http.ResponseWriter, r *http.Request) {
	examID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		middleware.WriteError(w, apperrors.BadRequest("invalid exam id"))
		return
	}
	userID, err := callerID(r)
	if err != nil {
		middleware.WriteError(w, apperrors.Internal(err))
		return
	}

	resp, err := h.svc.GetExam(r.Context(), userID, examID)
	if err != nil {
		middleware.WriteError(w, err)
		return
	}

	middleware.WriteJSON(w, http.StatusOK, resp)
}
