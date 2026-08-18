package dto

import (
	"time"

	"github.com/google/uuid"
)

// ExamListItem is one row of a teacher/school_admin's exam list (backs
// GET /api/v1/exams) -- lighter than ExamStatus: no parse_error/unit
// numbers/validation report, since the list view only needs enough to
// show status at a glance and link into the full exam detail screen.
type ExamListItem struct {
	ExamID          uuid.UUID `json:"examId"`
	Title           string    `json:"title"`
	SubjectCode     string    `json:"subjectCode"`
	GradeLevel      int       `json:"gradeLevel"`
	Status          string    `json:"status"`
	ExamScope       string    `json:"examScope"`
	TotalMarks      int       `json:"totalMarks"`
	QuestionCount   int       `json:"questionCount"`
	SubmissionCount int       `json:"submissionCount"`
	CreatedAt       time.Time `json:"createdAt"`
}
