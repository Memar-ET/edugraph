package dto

import (
	"time"

	"github.com/google/uuid"
)

// CloseExamResponse is returned by POST /exams/:id/close.
type CloseExamResponse struct {
	ExamID  uuid.UUID `json:"examId"`
	Status  string    `json:"status"`
	Message string    `json:"message"`
}

// AutosaveDraftAnswer is one answer entry in an autosave request. Response
// is a plain string (an option letter or free text) since answer options
// live as JSONB on assessment.questions, not a normalized options table.
type AutosaveDraftAnswer struct {
	QuestionID uuid.UUID `json:"questionId" validate:"required"`
	Response   string    `json:"response"`
}

// AutosaveRequest is the body for POST /exams/:id/autosave.
type AutosaveRequest struct {
	Answers []AutosaveDraftAnswer `json:"answers" validate:"required,min=1"`
}

// DraftAnswerItem is one saved draft entry returned by GET /exams/:id/draft.
type DraftAnswerItem struct {
	QuestionID uuid.UUID `json:"questionId"`
	Response   string    `json:"response"`
	SavedAt    time.Time `json:"savedAt"`
}

// ExamDraftResponse is the body for GET /exams/:id/draft.
type ExamDraftResponse struct {
	ExamID  uuid.UUID         `json:"examId"`
	Answers []DraftAnswerItem `json:"answers"`
	SavedAt time.Time         `json:"savedAt"`
}
