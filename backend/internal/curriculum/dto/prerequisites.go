package dto

// AddPrerequisiteRequest links a topic to one it depends on --
// "topic {id} requires prerequisiteTopicId first". Weight (0..1,
// default 1.0) expresses how essential the dependency is.
type AddPrerequisiteRequest struct {
	PrerequisiteTopicID string   `json:"prerequisiteTopicId" validate:"required,uuid"`
	Weight              *float64 `json:"weight" validate:"omitempty,gte=0,lte=1"`
}

// PrerequisiteLink is one edge of the prerequisite graph, titles resolved
// for display. IsCrossGrade is derived (grade levels differ), never
// client-supplied.
type PrerequisiteLink struct {
	TopicID             string  `json:"topicId"`
	TopicTitle          string  `json:"topicTitle"`
	PrerequisiteTopicID string  `json:"prerequisiteTopicId"`
	PrerequisiteTitle   string  `json:"prerequisiteTitle"`
	PrerequisiteGrade   int     `json:"prerequisiteGrade"`
	Weight              float64 `json:"weight"`
	IsCrossGrade        bool    `json:"isCrossGrade"`
	InferMethod         string  `json:"inferMethod"`
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
