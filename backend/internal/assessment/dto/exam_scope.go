package dto

import "github.com/google/uuid"

// UpdateExamScopeRequest lets a teacher correct the subject/grade/exam-type/
// unit-range that Capability 2A's title parser (deriveGradeAndScope) or the
// document's own metadata table got wrong -- previously the only fix was
// re-uploading the whole file with a better title. Every field is a
// pointer/nil-able slice so only the fields actually supplied are changed;
// omitted fields keep their current value. At least one field is required.
type UpdateExamScopeRequest struct {
	SubjectCode *string `json:"subjectCode,omitempty" validate:"omitempty"`
	GradeLevel  *int    `json:"gradeLevel,omitempty" validate:"omitempty,min=1,max=12"`
	// ExamScope mirrors title_parser.go's deriveExamScope output values.
	ExamScope *string `json:"examScope,omitempty" validate:"omitempty,oneof=unit_test midterm final_exam"`
	// UnitNumbers replaces the exam's unit_numbers wholesale when
	// supplied (e.g. [3] or [1,2,3]) -- send the full corrected list, not
	// a diff. An explicit empty array ([]) clears it (final_exam-style
	// "all units"); omit the field entirely to leave it unchanged.
	UnitNumbers []int `json:"unitNumbers,omitempty" validate:"omitempty,dive,min=1"`
}

type UpdateExamScopeResponse struct {
	ExamID      uuid.UUID `json:"examId"`
	SubjectCode string    `json:"subjectCode"`
	GradeLevel  int       `json:"gradeLevel"`
	ExamScope   string    `json:"examScope"`
	UnitNumbers []int     `json:"unitNumbers"`
	// CloRematchQueued is true when subjectCode or gradeLevel changed --
	// every question's existing CLO match was computed against the *old*
	// subject's CLO set and is now stale, so a background rematch
	// (queue:exam:rematch, see ai-service/app/workers/exam_rematch_worker.py)
	// is queued automatically rather than requiring a re-upload.
	CloRematchQueued bool   `json:"cloRematchQueued"`
	Message          string `json:"message"`
}
