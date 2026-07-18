package dto

import "time"

// Capability 4C: School Quality Scoring -- one composite 0..100 health
// score per subject+grade, replacing subjective evaluation with the
// weighted formula from the impl plan:
//
//	0.30 × clo_coverage_pct + 0.25 × student_mastery_pct
//	+ 0.25 × exam_quality_avg + 0.20 × curriculum_compliance
type SchoolQualityScore struct {
	SubjectCode string `json:"subjectCode"`
	GradeLevel  int    `json:"gradeLevel"`

	CLOCoveragePct       float64 `json:"cloCoveragePct"`
	StudentMasteryPct    float64 `json:"studentMasteryPct"`
	ExamQualityAvg       float64 `json:"examQualityAvg"`
	CurriculumCompliance float64 `json:"curriculumCompliance"`

	CompositeScore float64 `json:"compositeScore"`
	// Low curriculum compliance flags the school for Ministry review --
	// mandatory sections are being skipped.
	FlaggedForReview bool      `json:"flaggedForReview"`
	ComputedAt       time.Time `json:"computedAt"`
}

type SchoolQualityResponse struct {
	SchoolID string `json:"schoolId"`
	// "redis" when served from the 1h cache, "postgres" otherwise.
	Source string               `json:"source"`
	Scores []SchoolQualityScore `json:"scores"`
}
