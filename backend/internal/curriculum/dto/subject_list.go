package dto

import (
	"time"
)

// SubjectListItem is one row of the system-wide curriculum browser (backs
// the Ministry "curriculum by subject" list) -- GET /api/v1/curriculum/subjects.
// Unlike JobListItem (one officer's own upload history), this spans every
// promoted subject regardless of who uploaded it.
type SubjectListItem struct {
	Code                string     `json:"code"`
	NameEn              string     `json:"nameEn"`
	NameAm              *string    `json:"nameAm,omitempty"`
	GradeLevel          int        `json:"gradeLevel"`
	AcademicYear        string     `json:"academicYear"`
	MoeCode             *string    `json:"moeCode,omitempty"`
	IsMandatory         bool       `json:"isMandatory"`
	Version             int        `json:"version"`
	IsCurrent           bool       `json:"isCurrent"`
	PreviousVersionCode *string    `json:"previousVersionCode,omitempty"`
	FileName            *string    `json:"fileName,omitempty"`
	UploadedByName      *string    `json:"uploadedByName,omitempty"`
	ApprovedAt          *time.Time `json:"approvedAt,omitempty"`
	CreatedAt           time.Time  `json:"createdAt"`
	UnitCount           int        `json:"unitCount"`
	TopicCount          int        `json:"topicCount"`
	SubtopicCount       int        `json:"subtopicCount"`
	CloCount            int        `json:"cloCount"`
}
