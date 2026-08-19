package dto

// WeakTopic is one of a school's worst-performing curriculum topics.
type WeakTopic struct {
	TopicID         string  `json:"topic_id"`
	TopicTitle      string  `json:"topic_title"`
	AvgMastery      float64 `json:"avg_mastery"`
	AffectedStudents int64  `json:"affected_students"`
}

// UnderperformingSchool summarises a school that is below regional average.
type UnderperformingSchool struct {
	SchoolID          string      `json:"school_id"`
	SchoolName        string      `json:"school_name"`
	SchoolCode        string      `json:"school_code"`
	QualityScore      float64     `json:"quality_score"`
	MasteryRate       float64     `json:"mastery_rate"`
	FlaggedTopicsCount int64      `json:"flagged_topics_count"`
	TopWeakTopics     []WeakTopic `json:"top_weak_topics"`
}

// UnderperformingResponse is the envelope payload returned by the handler.
type UnderperformingResponse struct {
	Schools []UnderperformingSchool `json:"schools"`
}
