package dto

import "time"

// SkillState is one students.skill_states row for the authenticated
// student's own skill-map view (GET /students/me/skill-states) -- every
// topic they have any evidence for, not the single-topic detail Explain
// returns.
type SkillState struct {
	TopicID            string    `json:"topicId"`
	TopicTitle         string    `json:"topicTitle"`
	SubjectCode        string    `json:"subjectCode"`
	GradeLevel         int       `json:"gradeLevel"`
	MasteryProbability *float64  `json:"masteryProbability,omitempty"`
	MasteryStatus      string    `json:"masteryStatus"`
	Uncertainty        *float64  `json:"uncertainty,omitempty"`
	Trend              *string   `json:"trend,omitempty"`
	EvidenceCount      int       `json:"evidenceCount"`
	ForgettingRisk     *float64  `json:"forgettingRisk,omitempty"`
	UpdatedAt          time.Time `json:"updatedAt"`
}

type SkillStatesResponse struct {
	StudentID   string       `json:"studentId"`
	SkillStates []SkillState `json:"skillStates"`
}
