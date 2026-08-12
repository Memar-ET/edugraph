package dto

// GraphNode is one node in a subject's Neo4j knowledge graph, shaped for
// direct consumption by a frontend graph-visualization library (nodes +
// edges), not a database-shaped export.
type GraphNode struct {
	ID          string `json:"id"`
	Label       string `json:"label"`
	Type        string `json:"type"` // "subject" | "unit" | "topic" | "clo"
	GradeLevel  *int   `json:"gradeLevel,omitempty"`
	IsSubtopic  bool   `json:"isSubtopic,omitempty"`
	SubjectCode string `json:"subjectCode,omitempty"`
	// External marks a topic that belongs to a *different* subject than
	// the one requested -- the other end of a cross-grade prerequisite
	// edge (e.g. viewing BIO-G9's graph, one of its topics requires a
	// BIO-G7 topic). Rendered as a minimal node, not expanded further.
	External bool `json:"external,omitempty"`
}

type GraphEdge struct {
	Source string `json:"source"`
	Target string `json:"target"`
	Type   string `json:"type"` // "HAS_UNIT" | "HAS_TOPIC" | "HAS_SUBTOPIC" | "HAS_CLO" | "HAS_PREREQUISITE"
}

// SubjectGraph is the response of GET /curriculum/subjects/{code}/graph.
type SubjectGraph struct {
	Nodes []GraphNode `json:"nodes"`
	Edges []GraphEdge `json:"edges"`
}
