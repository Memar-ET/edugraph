package dto

// AddItemSkillMappingRequest records one Q-matrix entry (spec section
// 6.3): "question {id} assesses topicId with this relevance." Mirrors
// AddPrerequisiteRequest's shape -- see that file's doc comment for the
// weight-vs-confidence distinction this DTO's relevance/qmatrixConfidence
// pair follows the same logic for (relevance = how much the item draws on
// this skill; qmatrixConfidence = how sure we are of that mapping at all,
// left for a calibration pass to fill in -- never client-supplied at
// creation).
type AddItemSkillMappingRequest struct {
	TopicID          string  `json:"topicId" validate:"required,uuid"`
	CloCode          *string `json:"cloCode" validate:"omitempty"`
	Relevance        float64 `json:"relevance" validate:"required,gte=0,lte=1"`
	CognitiveLevel   *string `json:"cognitiveLevel" validate:"omitempty,oneof=remember understand apply analyze evaluate create"`
	GenerationMethod string  `json:"generationMethod" validate:"omitempty,oneof=manual ai_draft embedding_auto irt_calibrated teacher_confirmed"`
}

// ItemSkillMapping is one versioned Q-matrix row, titles resolved for
// display. Difficulty/discrimination/qmatrixConfidence are nil until a
// calibration pass (Milestone 8) populates them -- never fabricated at
// authoring time (spec section 15's cold-start principle).
type ItemSkillMapping struct {
	ID                string   `json:"id"`
	QuestionID        string   `json:"questionId"`
	TopicID           string   `json:"topicId"`
	TopicTitle        string   `json:"topicTitle"`
	CloCode           *string  `json:"cloCode,omitempty"`
	Relevance         float64  `json:"relevance"`
	CognitiveLevel    *string  `json:"cognitiveLevel,omitempty"`
	Difficulty        *float64 `json:"difficulty,omitempty"`
	Discrimination    *float64 `json:"discrimination,omitempty"`
	QMatrixConfidence *float64 `json:"qmatrixConfidence,omitempty"`
	GenerationMethod  string   `json:"generationMethod"`
	Version           int      `json:"version"`
	IsCurrent         bool     `json:"isCurrent"`
	IsValidated       bool     `json:"isValidated"`
}

// AddItemSkillMappingResponse reports the created mapping plus whether
// the best-effort Neo4j mirror succeeded -- same convention as
// AddPrerequisiteResponse.
type AddItemSkillMappingResponse struct {
	Mapping     ItemSkillMapping `json:"mapping"`
	GraphSynced bool             `json:"graphSynced"`
	GraphError  string           `json:"graphError,omitempty"`
}

// ResyncItemSkillMappingsResponse reports the outcome of a bulk Postgres
// -> Neo4j Q-matrix resync, mirroring ResyncPrerequisitesResponse.
type ResyncItemSkillMappingsResponse struct {
	Synced int `json:"synced"`
	Failed int `json:"failed"`
}
