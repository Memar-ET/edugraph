package dto

import (
	"time"

	"github.com/google/uuid"
)

// ExamAvailability is one row of the authenticated student's available-
// exams list (GET /api/v1/students/me/available-exams) -- every published
// exam matching their school/grade, same scoping verifyStudentAccess
// already enforces per-exam, plus whether they've already submitted it.
type ExamAvailability struct {
	ExamID           uuid.UUID  `json:"examId"`
	Title            string     `json:"title"`
	SubjectCode      string     `json:"subjectCode"`
	GradeLevel       int        `json:"gradeLevel"`
	ExamScope        string     `json:"examScope"`
	TotalMarks       int        `json:"totalMarks"`
	QuestionCount    int        `json:"questionCount"`
	PublishedAt      time.Time  `json:"publishedAt"`
	ClosesAt         *time.Time `json:"closesAt,omitempty"`
	AlreadyAttempted bool       `json:"alreadyAttempted"`
}

type AvailableExamsResponse struct {
	Exams []ExamAvailability `json:"exams"`
}
