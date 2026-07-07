package handler

import (
	"net/http"
	"strconv"

	"github.com/edugraph-ai/edugraph/internal/curriculum/dto"
	"github.com/edugraph-ai/edugraph/internal/curriculum/service"
	apperrors "github.com/edugraph-ai/edugraph/pkg/errors"
	"github.com/edugraph-ai/edugraph/pkg/middleware"
	"github.com/edugraph-ai/edugraph/pkg/validator"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

// maxUploadBytes bounds the in-memory portion of the multipart parse.
// Files larger than this spill to temp files on disk (handled by Go's
// multipart reader); the hard ceiling on the file itself is enforced by
// the reverse proxy / server ReadTimeout in production.
const maxUploadBytes = 25 << 20 // 25 MB

type Handler struct {
	service *service.Service
}

func New(service *service.Service) *Handler {
	return &Handler{service: service}
}

// Upload handles POST /api/v1/curriculum/upload
// Expects multipart/form-data with a "file" part (PDF or DOCX) plus the
// form fields "subjectCode", "gradeLevel", and "academicYear". Requires an
// authenticated user with the curriculum_officer or ministry_admin role
// (enforced by middleware.RequireRole in the router).
func (h *Handler) Upload(w http.ResponseWriter, r *http.Request) {
	// 1. Get the authenticated user ID set by the Authenticate middleware.
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

	// 2. Parse the multipart form. This reads the whole request body once,
	// so form fields and the file must both be read from it below — the
	// body cannot also be JSON-decoded afterwards.
	if err := r.ParseMultipartForm(maxUploadBytes); err != nil {
		middleware.WriteError(w, apperrors.BadRequest("invalid multipart form data"))
		return
	}

	gradeLevel, err := strconv.Atoi(r.FormValue("gradeLevel"))
	if err != nil {
		middleware.WriteError(w, apperrors.BadRequest("gradeLevel must be a number between 1 and 12"))
		return
	}

	req := dto.UploadRequest{
		SubjectCode:  r.FormValue("subjectCode"),
		GradeLevel:   gradeLevel,
		AcademicYear: r.FormValue("academicYear"),
	}
	if err := validator.Struct(req); err != nil {
		middleware.WriteError(w, apperrors.BadRequest(err.Error()))
		return
	}

	// 3. Get the uploaded file.
	file, header, err := r.FormFile("file")
	if err != nil {
		middleware.WriteError(w, apperrors.BadRequest("file is required"))
		return
	}
	defer file.Close()

	// Security check: only allow PDF or DOCX curriculum documents.
	mimeType := header.Header.Get("Content-Type")
	if mimeType != "application/pdf" &&
		mimeType != "application/vnd.openxmlformats-officedocument.wordprocessingml.document" {
		middleware.WriteError(w, apperrors.BadRequest("only PDF and DOCX files are allowed"))
		return
	}

	// 4. Hand off to the service: saves the file via the active
	// StorageProvider (Postgres in dev, S3 in prod), creates the upload_job
	// row, and queues the parse job for the AI service via Redis.
	resp, err := h.service.Upload(
		r.Context(),
		userID,
		req,
		header.Filename,
		mimeType,
		header.Size,
		file,
	)
	if err != nil {
		middleware.WriteError(w, apperrors.Internal(err))
		return
	}

	middleware.WriteJSON(w, http.StatusAccepted, resp)
}

// GetJob handles GET /api/v1/curriculum/jobs/:id
func (h *Handler) GetJob(w http.ResponseWriter, r *http.Request) {
	jobIDStr := chi.URLParam(r, "id")
	jobID, err := uuid.Parse(jobIDStr)
	if err != nil {
		middleware.WriteError(w, apperrors.BadRequest("invalid job id"))
		return
	}

	job, err := h.service.GetJob(r.Context(), jobID)
	if err != nil {
		middleware.WriteError(w, apperrors.NotFound("job not found"))
		return
	}

	middleware.WriteJSON(w, http.StatusOK, job)
}

// CreateUnit handles POST /api/v1/curriculum/units
func (h *Handler) CreateUnit(w http.ResponseWriter, r *http.Request) {
	middleware.WriteError(w, apperrors.NotImplemented("not yet implemented"))
}

// UpdateUnit handles PATCH /api/v1/curriculum/units/:id
func (h *Handler) UpdateUnit(w http.ResponseWriter, r *http.Request) {
	middleware.WriteError(w, apperrors.NotImplemented("not yet implemented"))
}

// DeleteUnit handles DELETE /api/v1/curriculum/units/:id
func (h *Handler) DeleteUnit(w http.ResponseWriter, r *http.Request) {
	middleware.WriteError(w, apperrors.NotImplemented("not yet implemented"))
}

// AddPrerequisite handles POST /api/v1/curriculum/units/:id/prerequisites
func (h *Handler) AddPrerequisite(w http.ResponseWriter, r *http.Request) {
	middleware.WriteError(w, apperrors.NotImplemented("not yet implemented"))
}
