package dto

// QMatrixQualityReport answers the EG-GCKT checklist's "detect items
// with missing or low-confidence skill mappings" / "flag weak or
// ambiguous Q-matrix mappings" for one subject.
type QMatrixQualityReport struct {
	SubjectCode           string                 `json:"subjectCode"`
	TotalQuestions        int                    `json:"totalQuestions"`
	MissingMappings       []QuestionMappingIssue `json:"missingMappings"`
	LowConfidenceMappings []QuestionMappingIssue `json:"lowConfidenceMappings"`
	AmbiguousMappings     []QuestionMappingIssue `json:"ambiguousMappings"`
}

// QuestionMappingIssue is one flagged question, with enough context for
// a curriculum officer to act on it without a second lookup.
type QuestionMappingIssue struct {
	QuestionID   string   `json:"questionId"`
	QuestionText string   `json:"questionText"`
	ExamID       string   `json:"examId"`
	Reason       string   `json:"reason"`
	TopicTitles  []string `json:"topicTitles,omitempty"`
}

// PrerequisiteQualityReport answers "run structural validation for
// orphaned skills... and duplicate edges" (duplicates are already
// impossible at the DB constraint level, so this focuses on what a
// constraint can't catch: isolation and low-confidence edges).
type PrerequisiteQualityReport struct {
	SubjectCode        string                   `json:"subjectCode"`
	TotalTopics        int                      `json:"totalTopics"`
	OrphanedTopics     []OrphanedTopic          `json:"orphanedTopics"`
	LowConfidenceEdges []LowConfidenceEdgeIssue `json:"lowConfidenceEdges"`
}

type OrphanedTopic struct {
	TopicID string `json:"topicId"`
	Title   string `json:"title"`
}

type LowConfidenceEdgeIssue struct {
	TopicID             string   `json:"topicId"`
	TopicTitle          string   `json:"topicTitle"`
	PrerequisiteTopicID string   `json:"prerequisiteTopicId"`
	PrerequisiteTitle   string   `json:"prerequisiteTitle"`
	EdgeType            string   `json:"edgeType"`
	Confidence          *float64 `json:"confidence,omitempty"`
	IsValidated         bool     `json:"isValidated"`
}
