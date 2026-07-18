package dto

import "github.com/google/uuid"

type UploadAnswerKeyResponse struct {
	ExamID  uuid.UUID `json:"examId"`
	Message string    `json:"message"`
}
