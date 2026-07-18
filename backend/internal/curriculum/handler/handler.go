package handler

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"

	"github.com/edugraph-ai/edugraph/internal/curriculum/dto"
	"github.com/edugraph-ai/edugraph/internal/curriculum/service"
	apperrors "github.com/edugraph-ai/edugraph/pkg/errors"
	"github.com/edugraph-ai/edugraph/pkg/fileformat"
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

	// Security check: sniff the actual file content (magic bytes), not the
	// client-supplied Content-Type header -- that header is attacker-controlled
	// and trivially spoofed. See architecture doc §9.6: "MIME type validated
	// server-side (magic bytes), not by Content-Type header."
	mimeType, ok := fileformat.SniffPDFOrDOCX(file)
	if !ok {
		middleware.WriteError(w, apperrors.BadRequest("file content is not a valid PDF or DOCX document"))
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
// Returns the job's current status and, once parsing has completed, the
// AI-parsed tree in parsedStructure -- this is what the frontend's review
// screen (Step 3) fetches to render "Unit 1: Mechanics -> Topic 1.1: Kinematics".
func (h *Handler) GetJob(w http.ResponseWriter, r *http.Request) {
	jobIDStr := chi.URLParam(r, "id")
	jobID, err := uuid.Parse(jobIDStr)
	if err != nil {
		middleware.WriteError(w, apperrors.BadRequest("invalid job id"))
		return
	}

	job, err := h.service.GetJob(r.Context(), jobID)
	if err != nil {
		writeServiceError(w, err)
		return
	}

	middleware.WriteJSON(w, http.StatusOK, job)
}

// DownloadFile handles GET /api/v1/storage/files/{jobId}
//
// Dev-mode half of Step 3's "Fork in the Road": streams the originally
// uploaded file (from Postgres BYTEA via the active StorageProvider) so the
// curriculum officer can view it alongside the parsed tree. In prod mode,
// swapping the StorageProvider for an S3-backed one that returns a
// presigned URL instead is the only change needed -- this handler doesn't
// know or care which one is active.
func (h *Handler) DownloadFile(w http.ResponseWriter, r *http.Request) {
	jobIDStr := chi.URLParam(r, "jobId")
	jobID, err := uuid.Parse(jobIDStr)
	if err != nil {
		middleware.WriteError(w, apperrors.BadRequest("invalid job id"))
		return
	}

	reader, fileName, mimeType, err := h.service.DownloadFile(r.Context(), jobID)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	defer reader.Close()

	w.Header().Set("Content-Type", mimeType)
	w.Header().Set("Content-Disposition", fmt.Sprintf("inline; filename=%q", fileName))
	w.WriteHeader(http.StatusOK)
	_, _ = io.Copy(w, reader) // headers are already sent; nothing useful to do with a copy error here
}

// Approve handles POST /api/v1/curriculum/jobs/{id}/approve
//
// The curriculum officer has reviewed the AI-parsed tree (and possibly
// corrected it in the UI) and is ready to make it the source of truth.
// The request body is optional: if the officer edited anything, the
// frontend sends the complete corrected tree as {"parsedStructure": {...}};
// otherwise an empty body approves the tree exactly as parsed.
//
// On success, every unit/topic/CLO in the approved tree is promoted into
// curriculum.units/topics/clos (upserted, so re-approving after further
// edits updates rather than duplicates), and the job is marked 'approved'.
func (h *Handler) Approve(w http.ResponseWriter, r *http.Request) {
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

	jobIDStr := chi.URLParam(r, "id")
	jobID, err := uuid.Parse(jobIDStr)
	if err != nil {
		middleware.WriteError(w, apperrors.BadRequest("invalid job id"))
		return
	}

	var req dto.ApproveRequest
	if r.ContentLength != 0 {
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			middleware.WriteError(w, apperrors.BadRequest("invalid JSON body"))
			return
		}
		if req.ParsedStructure != nil {
			if err := validator.Struct(*req.ParsedStructure); err != nil {
				middleware.WriteError(w, apperrors.BadRequest(err.Error()))
				return
			}
		}
	}

	resp, err := h.service.Approve(r.Context(), userID, jobID, req.ParsedStructure)
	if err != nil {
		writeServiceError(w, err)
		return
	}

	middleware.WriteJSON(w, http.StatusOK, resp)
}

// writeServiceError unwraps an *apperrors.AppError produced by the service
// layer (NotFound, Conflict, BadRequest, ...) so its real status code
// reaches the client, instead of collapsing every failure into a generic one.
func writeServiceError(w http.ResponseWriter, err error) {
	if appErr, ok := apperrors.As(err); ok {
		middleware.WriteError(w, appErr)
		return
	}
	middleware.WriteError(w, apperrors.Internal(err))
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
