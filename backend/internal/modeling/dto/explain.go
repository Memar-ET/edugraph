package dto

import "time"

// Explanation is the EG-GCKT spec section 18 five-part explanation for
// one (student, topic) pair: what was observed, what evidence supported
// it, what graph relationships influenced the conclusion, how confident
// the system is, and what it recommends next (and why).
type Explanation struct {
	StudentID         string                       `json:"studentId"`
	TopicID           string                       `json:"topicId"`
	TopicTitle        string                       `json:"topicTitle"`
	CurrentState      ExplanationCurrentState      `json:"currentState"`
	Evidence          []ExplanationEvidenceItem    `json:"evidence"`
	StructuralContext ExplanationStructuralContext `json:"structuralContext"`
	Confidence        string                       `json:"confidence"`
	Recommendation    string                       `json:"recommendation"`
	Reason            string                       `json:"reason"`
}

// ExplanationCurrentState mirrors the fields of students.skill_states
// most relevant to a human reader -- not every column (diagnostic_
// provenance/model_snapshot_id are implementation detail, not part of
// the human-facing explanation).
type ExplanationCurrentState struct {
	MasteryProbability *float64   `json:"masteryProbability,omitempty"`
	MasteryStatus      string     `json:"masteryStatus"`
	Trend              *string    `json:"trend,omitempty"`
	EvidenceCount      int        `json:"evidenceCount"`
	ForgettingRisk     *float64   `json:"forgettingRisk,omitempty"`
	LastSeen           *time.Time `json:"lastSeen,omitempty"`
}

// ExplanationEvidenceItem is one modeling.evidence_log row, the most
// recent per source (spec: "what evidence supported the estimate").
type ExplanationEvidenceItem struct {
	Provenance  string    `json:"provenance"`
	Estimate    *float64  `json:"estimate,omitempty"`
	Uncertainty *float64  `json:"uncertainty,omitempty"`
	Reliability *float64  `json:"reliability,omitempty"`
	CreatedAt   time.Time `json:"createdAt"`
}

// PrerequisiteMasterySummary is one direct prerequisite of the explained
// topic, with the student's current mastery on it -- the "what graph
// relationships influenced the conclusion" part of the spec's five-part
// explanation.
type PrerequisiteMasterySummary struct {
	TopicID            string   `json:"topicId"`
	Title              string   `json:"title"`
	EdgeType           string   `json:"edgeType"`
	MasteryProbability *float64 `json:"masteryProbability,omitempty"`
}

type ExplanationStructuralContext struct {
	Prerequisites []PrerequisiteMasterySummary `json:"prerequisites"`
}
