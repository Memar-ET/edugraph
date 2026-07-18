package dto

// Capability 4A: Class-Wide Gap Heatmap.

// HeatmapTopic is one cell of the heatmap: a topic ranked by how much of
// the teacher's class (same school + grade level) is struggling with it.
type HeatmapTopic struct {
	TopicID            string        `json:"topicId"`
	Title              string        `json:"title"`
	StudentsStruggling int           `json:"studentsStruggling"`
	StrugglingPct      float64       `json:"strugglingPct"`
	AvgSeverity        float64       `json:"avgSeverity"`
	Alert              *HeatmapAlert `json:"alert,omitempty"`
}

// HeatmapAlert is the cross-grade root-cause warning: fired when the
// struggling share crosses the alert threshold AND the prerequisite graph
// traces the failure to a lower-grade topic this cohort is also weak in.
type HeatmapAlert struct {
	TopicID            string  `json:"topicId"`
	TopicTitle         string  `json:"topicTitle"`
	StrugglingPct      float64 `json:"strugglingPct"`
	RootCauseTopicID   string  `json:"rootCauseTopicId"`
	RootCauseTitle     string  `json:"rootCauseTitle"`
	RootCauseGrade     int     `json:"rootCauseGradeLevel"`
	StudentsStruggling int     `json:"rootCauseStudentsStruggling"`
	Message            string  `json:"message"`
}

type ClassHeatmapResponse struct {
	SubjectCode string `json:"subjectCode"`
	GradeLevel  int    `json:"gradeLevel"`
	SchoolID    string `json:"schoolId"`
	ClassSize   int    `json:"classSize"`
	// Source records which store answered: "neo4j" (graph-first) or
	// "postgres" (fallback over students.gap_records).
	Source string         `json:"source"`
	Topics []HeatmapTopic `json:"topics"`
	Alerts []HeatmapAlert `json:"alerts"`
}
