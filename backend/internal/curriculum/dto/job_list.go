package dto

import (
	"time"

	"github.com/google/uuid"
)

// JobListItem is one row of a Curriculum Officer's upload/approval
// history (feature 1.2's dashboard) -- GET /api/v1/curriculum/jobs.
// Deliberately lighter than JobStatus: no parsed_structure, since a list
// view only needs enough to show status at a glance and link into the
// full review screen.
type JobListItem struct {
	JobID        uuid.UUID  `json:"jobId"`
	Status       string     `json:"status"`
	FileName     string     `json:"fileName"`
	SubjectCode  string     `json:"subjectCode"`
	GradeLevel   int        `json:"gradeLevel"`
	AcademicYear string     `json:"academicYear"`
	CreatedAt    time.Time  `json:"createdAt"`
	ApprovedAt   *time.Time `json:"approvedAt,omitempty"`
}
