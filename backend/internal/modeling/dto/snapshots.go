package dto

import "time"

// SkillStateSnapshot is one immutable point-in-time copy of a student's
// fused knowledge state for a topic (EG-GCKT checklist sections 6/18/22:
// "support learner-state snapshots," "support historical state
// comparison"). Distinct from the live students.skill_states row, which
// is overwritten in place -- a list of these lets a teacher/reviewer
// compare mastery over time.
type SkillStateSnapshot struct {
	ID                    string     `json:"id"`
	MasteryProbability    *float64   `json:"masteryProbability,omitempty"`
	MasteryStatus         string     `json:"masteryStatus"`
	Uncertainty           *float64   `json:"uncertainty,omitempty"`
	EvidenceCount         int        `json:"evidenceCount"`
	Trend                 *string    `json:"trend,omitempty"`
	SnapshotReason        string     `json:"snapshotReason"`
	SourceEventRangeStart *time.Time `json:"sourceEventRangeStart,omitempty"`
	SourceEventRangeEnd   *time.Time `json:"sourceEventRangeEnd,omitempty"`
	TakenAt               time.Time  `json:"takenAt"`
}
