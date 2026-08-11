package dto

// TopicListItem is one row of GET /api/v1/curriculum/subjects/{code}/topics
// -- a flat, subject-scoped topic list. Exists specifically to back topic
// lookup for the prerequisites UI: there's no general "browse the whole
// curriculum" endpoint, and the prerequisite endpoints need a topic UUID
// the frontend has no other way to discover. ParentTopicID lets the
// frontend indent a subtopic under its parent for readability, mirroring
// curriculum.topics.parent_topic_id (V027).
type TopicListItem struct {
	ID            string  `json:"id"`
	TitleEn       string  `json:"titleEn"`
	UnitNumber    int     `json:"unitNumber"`
	SequenceOrder int     `json:"sequenceOrder"`
	ParentTopicID *string `json:"parentTopicId,omitempty"`
}
