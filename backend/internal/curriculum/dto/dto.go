package dto

import "github.com/google/uuid"

// UploadRequest represents the multipart form fields sent by the frontend
// alongside the uploaded file. Fields are read manually from the request
// via r.FormValue in the handler (Go's stdlib has no form-tag decoder), so
// the "form" tags below document field names rather than driving decoding.
type UploadRequest struct {
	SubjectCode  string `form:"subjectCode" validate:"required"`
	GradeLevel   int    `form:"gradeLevel" validate:"required,min=1,max=12"`
	AcademicYear string `form:"academicYear" validate:"required"`
}

// UploadResponse is returned immediately after the job is queued.
type UploadResponse struct {
	JobID   uuid.UUID `json:"jobId"`
	Status  string    `json:"status"`
	Message string    `json:"message"`
}

// JobStatus represents the current state of the parsing pipeline.
type JobStatus struct {
	JobID    uuid.UUID `json:"jobId"`
	Status   string    `json:"status"` // pending, parsing, parsed, approved, failed
	FileName string    `json:"fileName"`
	Error    *string   `json:"error,omitempty"`
}
