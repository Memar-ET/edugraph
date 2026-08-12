package dto

import "time"

// SupersedeRequest is the body of POST /api/v1/curriculum/subjects/{code}/supersede.
// {code} is the already-approved subject that becomes the new current
// version; PreviousCode names the version it replaces.
type SupersedeRequest struct {
	PreviousCode string `json:"previousCode" validate:"required"`
}

// SubjectVersion is one entry in a subject's revision lineage (feature
// 1.3). Every version keeps its own subject_code and every row it owns
// (units/topics/clos) untouched forever -- superseding a version only
// flips IsCurrent/SupersededAt, it never rewrites or deletes anything, so
// existing gap_records/study_plans/exam questions that point at an older
// version's topic_id/clo_code keep meaning exactly what they meant when
// they were written.
type SubjectVersion struct {
	Code                string     `json:"code"`
	Version             int        `json:"version"`
	IsCurrent           bool       `json:"isCurrent"`
	AcademicYear        string     `json:"academicYear"`
	PreviousVersionCode *string    `json:"previousVersionCode,omitempty"`
	SupersededAt        *time.Time `json:"supersededAt,omitempty"`
	CreatedAt           time.Time  `json:"createdAt"`
}
