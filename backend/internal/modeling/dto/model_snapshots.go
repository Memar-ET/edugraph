package dto

import "time"

// ModelSnapshot is one row of modeling.model_snapshots (EG-GCKT Milestone
// 0's governance backbone) -- a versioned parameter set/policy for one
// engine (BKT, DINA, IRT, ...), with a candidate/validated/active/
// rejected/superseded lifecycle. Backs the ministry/curriculum-officer
// governance review queue (Milestone 9).
type ModelSnapshot struct {
	ID              string                 `json:"id"`
	ModelType       string                 `json:"modelType"`
	Version         int                    `json:"version"`
	Status          string                 `json:"status"`
	Scope           *string                `json:"scope,omitempty"`
	Config          map[string]interface{} `json:"config"`
	TrainingSummary map[string]interface{} `json:"trainingSummary,omitempty"`
	Notes           *string                `json:"notes,omitempty"`
	CreatedAt       time.Time              `json:"createdAt"`
}
