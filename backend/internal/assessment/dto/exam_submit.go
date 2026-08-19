package dto

import "github.com/google/uuid"

// AnswerInput: Response is the selected option letter for mcq, or free
// text for everything else. TimeSpentSecs is optional client-reported
// per-question time -- it feeds 4B's time-anomaly detection and is never
// used for grading.
type AnswerInput struct {
	QuestionID    uuid.UUID `json:"questionId" validate:"required"`
	Response      string    `json:"response"`
	TimeSpentSecs *int      `json:"timeSpentSecs" validate:"omitempty,gte=0"`
}

// IdempotencyKey is client-generated (one per attempt, see TakeExamPage's
// sessionStorage-backed key) so a double-click or a network-retried
// submit resolves to the same attempt instead of erroring or -- before
// this was wired up -- silently being accepted twice. See
// idx_exam_attempts_idempotency (V057) and Repository.SubmitAndFinalize.
type SubmitExamRequest struct {
	Answers        []AnswerInput `json:"answers" validate:"required,min=1,dive"`
	IdempotencyKey *uuid.UUID    `json:"idempotencyKey,omitempty"`
}

// SubmitExamResponse: TotalScore/Percentage/Passed are nil until every
// question on the attempt has been graded (auto or manual) -- see
// repository.RecomputeAttemptTotals.
type SubmitExamResponse struct {
	AttemptID           uuid.UUID `json:"attemptId"`
	GradedCount         int       `json:"gradedCount"`
	PendingGradingCount int       `json:"pendingGradingCount"`
	TotalScore          *float64  `json:"totalScore,omitempty"`
	Percentage          *float64  `json:"percentage,omitempty"`
	Passed              *bool     `json:"passed,omitempty"`
}

// GradeEntry: Value is an MCQ option letter (e.g. "B") when the question
// is mcq, or the teacher's already-graded numeric marks (e.g. "2") for
// everything else -- the paper was graded by hand before this was typed
// in, this isn't triggering a re-grade.
type GradeEntry struct {
	StudentID  uuid.UUID `json:"studentId" validate:"required"`
	QuestionID uuid.UUID `json:"questionId" validate:"required"`
	Value      string    `json:"value" validate:"required"`
}

type BulkGradeRequest struct {
	Entries []GradeEntry `json:"entries" validate:"required,min=1,dive"`
}

type BulkGradeResponse struct {
	AttemptsTouched int `json:"attemptsTouched"`
	AnswersSaved    int `json:"answersSaved"`
}
