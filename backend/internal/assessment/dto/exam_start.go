package dto

import (
	"time"

	"github.com/google/uuid"
)

// StartAttemptResponse is returned by POST /exams/{id}/start. Calling it
// again while an attempt is still in_progress is idempotent -- it returns
// the SAME attempt/question order/expiry rather than creating a new one
// or re-randomizing, which is what makes browser-refresh resume work
// without a separate "resume" endpoint.
type StartAttemptResponse struct {
	AttemptID        uuid.UUID         `json:"attemptId"`
	ExamID           uuid.UUID         `json:"examId"`
	AttemptNumber    int               `json:"attemptNumber"`
	StartedAt        time.Time         `json:"startedAt"`
	ExpiresAt        *time.Time        `json:"expiresAt,omitempty"`
	TimeLimitMinutes *int              `json:"timeLimitMinutes,omitempty"`
	Questions        []StudentQuestion `json:"questions"`
}
