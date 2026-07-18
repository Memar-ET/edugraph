package dto

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// GenerateStudyPlanRequest queues Capability 3B generation. TargetExamID
// scopes the plan to that exam's subject; omitted means "plan across all
// my unresolved gaps". Language is the student's preferred plan language.
type GenerateStudyPlanRequest struct {
	TargetExamID *string `json:"targetExamId" validate:"omitempty,uuid"`
	Language     *string `json:"language" validate:"omitempty,oneof=en am"`
}

// GenerateStudyPlanResponse: generation is asynchronous (the ai-service
// worker reads gap records + the prerequisite graph) -- the client polls
// GET /students/me/study-plans for the result.
type GenerateStudyPlanResponse struct {
	Status  string `json:"status"`
	Message string `json:"message"`
}

// StudyPlan is one students.study_plans row. PlanData is the day-by-day
// JSONB produced by the generator: {"days": [{"day", "blocks": [{
// "topicId", "title", "hours", "why", "isRootCause", ...}]}], "summary"}.
type StudyPlan struct {
	ID           uuid.UUID       `json:"id"`
	TargetExamID *uuid.UUID      `json:"targetExamId"`
	PlanData     json.RawMessage `json:"planData"`
	TotalDays    int             `json:"totalDays"`
	TotalHours   float64         `json:"totalHours"`
	Language     string          `json:"language"`
	GeneratedAt  time.Time       `json:"generatedAt"`
	ExpiresAt    *time.Time      `json:"expiresAt"`
}
