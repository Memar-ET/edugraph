package dto

import (
	"time"

	"github.com/google/uuid"
)

// MisconceptionHypothesis is one row of students.misconception_hypotheses
// (EG-GCKT Milestone 6, spec section 11) -- a structured, LLM-proposed
// candidate misconception the ai-service gap-analysis pipeline generated
// when it saw a repeated wrong-answer pattern on one topic, pending
// teacher review.
type MisconceptionHypothesis struct {
	ID                 uuid.UUID `json:"id"`
	StudentID          uuid.UUID `json:"studentId"`
	TopicID            uuid.UUID `json:"topicId"`
	TopicTitle         string    `json:"topicTitle"`
	MisconceptionText  string    `json:"misconceptionText"`
	TriggerPattern     *string   `json:"triggerPattern,omitempty"`
	Confidence         *float64  `json:"confidence,omitempty"`
	Status             string    `json:"status"`
	InterventionText   *string   `json:"interventionText,omitempty"`
	GeneratedByModel   *string   `json:"generatedByModel,omitempty"`
	CreatedAt          time.Time `json:"createdAt"`
}

// ReviewMisconceptionRequest confirms or rejects a candidate hypothesis --
// the only two valid teacher decisions (a hypothesis is never deleted, so
// a wrong LLM guess stays visible as evidence of what the system tried).
type ReviewMisconceptionRequest struct {
	Decision string `json:"decision" validate:"required,oneof=confirmed rejected"`
}
