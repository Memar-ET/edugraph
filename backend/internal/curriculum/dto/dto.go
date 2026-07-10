package dto

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

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

// JobStatus represents the current state of the parsing pipeline, including
// the AI-parsed tree once available -- this is what the frontend's review
// screen (Step 3) fetches via GET /api/v1/curriculum/jobs/{id}.
type JobStatus struct {
	JobID           uuid.UUID       `json:"jobId"`
	Status          string          `json:"status"` // pending, parsing, parsed, review, approved, rejected, failed
	FileName        string          `json:"fileName"`
	SubjectCode     string          `json:"subjectCode"`
	GradeLevel      int             `json:"gradeLevel"`
	AcademicYear    string          `json:"academicYear"`
	ParsedStructure json.RawMessage `json:"parsedStructure,omitempty"`
	ApprovedBy      *uuid.UUID      `json:"approvedBy,omitempty"`
	ApprovedAt      *time.Time      `json:"approvedAt,omitempty"`
	Error           *string         `json:"error,omitempty"`
}

// ---------------------------------------------------------------------------
// Human review / approval (Step 3)
// ---------------------------------------------------------------------------

// ParsedCLO mirrors a curriculum.clos row plus the mapping fields needed to
// link it to a topic. Matches the shape the AI service writes into
// parsed_structure (see ai-service/app/services/curriculum_parser).
type ParsedCLO struct {
	Code        string `json:"code" validate:"required"`
	Description string `json:"description" validate:"required"`
	BloomLevel  string `json:"bloomLevel"`
	Mandatory   bool   `json:"mandatory"`
	KeyConcept  string `json:"keyConcept"`
	Evidence    string `json:"evidence"`
}

// ParsedTopic mirrors a curriculum.topics row.
type ParsedTopic struct {
	SequenceOrder    int         `json:"sequenceOrder"`
	TitleEn          string      `json:"titleEn" validate:"required"`
	KeyConcepts      []string    `json:"keyConcepts"`
	LearningOutcomes []string    `json:"learningOutcomes"`
	Clos             []ParsedCLO `json:"clos"`
	RawText          string      `json:"rawText"`
}

// ParsedUnit mirrors a curriculum.units row.
type ParsedUnit struct {
	Number  int           `json:"number" validate:"required"`
	TitleEn string        `json:"titleEn" validate:"required"`
	Topics  []ParsedTopic `json:"topics"`
}

// ParsedStructurePayload is the full tree, matching upload_jobs.parsed_structure.
type ParsedStructurePayload struct {
	Units []ParsedUnit `json:"units" validate:"required,min=1,dive"`
}

// ApproveRequest is the body of POST /api/v1/curriculum/jobs/{id}/approve.
//
// ParsedStructure is optional: if the curriculum officer edited the AI's
// tree in the UI (fixed a merged chapter, added a missing subtopic, etc.),
// the frontend sends the corrected full tree here and it becomes both what
// gets promoted into curriculum.units/topics/clos AND what's persisted back
// into upload_jobs.parsed_structure. If omitted, the tree already stored
// from parsing is used as-is.
//
// IMPORTANT: this must be the *complete* tree, not a diff -- any unit/topic/
// CLO left out is treated as removed by the officer's edit.
type ApproveRequest struct {
	ParsedStructure *ParsedStructurePayload `json:"parsedStructure,omitempty"`
}

// ApproveResponse summarizes what got promoted into the real curriculum
// tables (Step 3) and whether the Knowledge Graph mirror succeeded (Step 4).
//
// GraphSynced is deliberately independent of the HTTP status: the approval
// itself (Postgres promotion) is the source of truth for the review
// workflow and can succeed even if the Neo4j sync fails. When GraphSynced
// is false, curriculum.upload_jobs.neo4j_written stays false too, and
// simply calling approve again (safe/idempotent) retries the graph sync.
type ApproveResponse struct {
	JobID          uuid.UUID `json:"jobId"`
	Status         string    `json:"status"`
	SubjectCode    string    `json:"subjectCode"`
	UnitsPromoted  int       `json:"unitsPromoted"`
	TopicsPromoted int       `json:"topicsPromoted"`
	ClosPromoted   int       `json:"closPromoted"`
	GraphSynced    bool      `json:"graphSynced"`
	GraphSyncError string    `json:"graphSyncError,omitempty"`
}
