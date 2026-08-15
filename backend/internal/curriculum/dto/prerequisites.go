package dto

// AddPrerequisiteRequest links a topic to one it depends on --
// "topic {id} requires prerequisiteTopicId first". Weight (0..1,
// default 1.0) expresses how essential the dependency is; Confidence
// (0..1, EG-GCKT addition) is a distinct, orthogonal notion -- how sure
// we are the edge is correct at all. An AI-inferred edge can claim
// weight=0.9 (strongly essential, if true) with confidence=0.3 (the model
// wasn't sure) at the same time.
//
// InferMethod defaults to "manual" (a human typed this in, so it's
// auto-validated -- see Repository.AddTopicPrerequisite). Passing
// "ai_inferred" creates the link as validated=false, pending a separate
// PATCH .../validate call -- the "inferred vs. validated" distinction.
//
// EdgeType (EG-GCKT addition, spec section 6.2) defaults to "requires"
// -- the only edge meaning this table supported before. Only
// requires/strongly_requires participate in cycle detection and the
// prerequisite-chain traversals (topological sort, root-cause walk,
// class-heatmap cross-grade alert); the other five types are softer
// associations that must never hard-block or be treated as a dependency
// chain.
type AddPrerequisiteRequest struct {
	PrerequisiteTopicID string   `json:"prerequisiteTopicId" validate:"required,uuid"`
	Weight              *float64 `json:"weight" validate:"omitempty,gte=0,lte=1"`
	InferMethod         string   `json:"inferMethod" validate:"omitempty,oneof=explicit ai_inferred manual moe_document"`
	EdgeType            string   `json:"edgeType" validate:"omitempty,oneof=requires strongly_requires recommended_before related_to similar_to supports alternative_to"`
	Confidence          *float64 `json:"confidence" validate:"omitempty,gte=0,lte=1"`
	Evidence            *string  `json:"evidence"`
}

// PrerequisiteLink is one edge of the prerequisite graph, titles resolved
// for display. IsCrossGrade is derived (grade levels differ), never
// client-supplied. IsValidated mirrors confirmed_at IS NOT NULL: true for
// every link added through the normal (human-authored) path, false for an
// "ai_inferred" link that hasn't been confirmed via PATCH .../validate yet.
type PrerequisiteLink struct {
	TopicID             string   `json:"topicId"`
	TopicTitle          string   `json:"topicTitle"`
	PrerequisiteTopicID string   `json:"prerequisiteTopicId"`
	PrerequisiteTitle   string   `json:"prerequisiteTitle"`
	PrerequisiteGrade   int      `json:"prerequisiteGrade"`
	Weight              float64  `json:"weight"`
	IsCrossGrade        bool     `json:"isCrossGrade"`
	InferMethod         string   `json:"inferMethod"`
	IsValidated         bool     `json:"isValidated"`
	EdgeType            string   `json:"edgeType"`
	Confidence          *float64 `json:"confidence,omitempty"`
	Evidence            *string  `json:"evidence,omitempty"`
	CreatedBy           *string  `json:"createdBy,omitempty"`
	CreatedByModel      *string  `json:"createdByModel,omitempty"`
	CurriculumVersion   *string  `json:"curriculumVersion,omitempty"`
}

// PrerequisiteReviewHistoryEntry is one row of
// curriculum.prerequisite_review_history -- backs
// GET /curriculum/topics/:id/prerequisites/:prereqId/history (Milestone 11).
type PrerequisiteReviewHistoryEntry struct {
	ID              string  `json:"id"`
	Action          string  `json:"action"`
	PreviousValues  *string `json:"previousValues,omitempty"`
	NewValues       *string `json:"newValues,omitempty"`
	ReviewedBy      *string `json:"reviewedBy,omitempty"`
	ReviewedAt      string  `json:"reviewedAt"`
	Notes           *string `json:"notes,omitempty"`
}

// AddPrerequisiteResponse reports the created link plus whether the
// best-effort Neo4j mirror succeeded (same convention as curriculum
// approval's graph sync -- Postgres is the system of record, a failed
// graph write degrades analysis, it doesn't fail the request).
type AddPrerequisiteResponse struct {
	Link        PrerequisiteLink `json:"link"`
	GraphSynced bool             `json:"graphSynced"`
	GraphError  string           `json:"graphError,omitempty"`
}

// ResyncPrerequisitesResponse reports the outcome of a bulk Postgres ->
// Neo4j prerequisite resync (feature 1.5) -- catches up any link whose
// best-effort sync failed at creation/validation time, or that predates
// the neo4j_written column (V026) entirely.
type ResyncPrerequisitesResponse struct {
	Synced int `json:"synced"`
	Failed int `json:"failed"`
}
