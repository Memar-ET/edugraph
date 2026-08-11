package dto

import (
	"time"

	"github.com/google/uuid"
)

// UploadExamRequest deliberately has no subjectCode/gradeLevel/examScope
// fields -- those are derived server-side from the title (see
// service.deriveGradeAndScope and repository.MatchSubjectFromTitle) so the
// title is the single source of truth for what curriculum the exam covers,
// e.g. "Grade 11 Biology Unit Test - Cell Biology".
type UploadExamRequest struct {
	Title        string `form:"title" validate:"required"`
	AcademicYear string `form:"academicYear" validate:"required"`
	Term         *int   `form:"term" validate:"omitempty,min=1,max=4"`
	TotalMarks   int    `form:"totalMarks" validate:"required,min=1"`
}

type UploadExamResponse struct {
	ExamID  uuid.UUID `json:"examId"`
	Status  string    `json:"status"`
	Message string    `json:"message"`
}

// ExamStatus backs GET /api/v1/exams/:id -- the teacher-facing equivalent
// of curriculum's JobStatus, polled while parsing is in progress.
type ExamStatus struct {
	ExamID        uuid.UUID `json:"examId"`
	Status        string    `json:"status"`
	Title         string    `json:"title"`
	SubjectCode   string    `json:"subjectCode"`
	GradeLevel    int       `json:"gradeLevel"`
	ExamScope     string    `json:"examScope"`
	AcademicYear  string    `json:"academicYear"`
	TotalMarks    int       `json:"totalMarks"`
	QuestionCount int       `json:"questionCount"`
	ParseError    *string   `json:"parseError,omitempty"`
	CreatedAt     time.Time `json:"createdAt"`
	// ValidationReport is nil until Capability 2B's POST .../validate has
	// been called at least once -- GET surfaces the last computed report so
	// there's no separate GET-report endpoint to keep in sync.
	ValidationReport *ValidationReport `json:"validationReport,omitempty"`
}
