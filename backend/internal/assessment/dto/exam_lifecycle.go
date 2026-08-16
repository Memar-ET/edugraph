package dto

import "github.com/google/uuid"

// CloseExamResponse is returned by POST /exams/:id/close.
type CloseExamResponse struct {
	ExamID  uuid.UUID `json:"examId"`
	Status  string    `json:"status"`
	Message string    `json:"message"`
}

// AutosaveDraftAnswer is one answer entry in an autosave request.
type AutosaveDraftAnswer struct {
	QuestionID       uuid.UUID  `json:"questionId" validate:"required"`
	SelectedOptionID *uuid.UUID `json:"selectedOptionId,omitempty"`
	TextAnswer       *string    `json:"textAnswer,omitempty"`
}

// AutosaveRequest is the body for POST /exams/:id/autosave.
type AutosaveRequest struct {
	Answers []AutosaveDraftAnswer `json:"answers" validate:"required,min=1"`
}

// DraftAnswerItem is one saved draft entry returned by GET /exams/:id/draft.
type DraftAnswerItem struct {
	QuestionID       uuid.UUID  `json:"questionId"`
	SelectedOptionID *uuid.UUID `json:"selectedOptionId,omitempty"`
	TextAnswer       *string    `json:"textAnswer,omitempty"`
}
