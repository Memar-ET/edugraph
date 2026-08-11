package dto

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// GapRecordEntry is one row of the Granular Layer (students.gap_records):
// a specific mistake traced Question -> CLO -> symptom Topic, plus where
// the prerequisite walk found the break in the chain (Capability 3A).
type GapRecordEntry struct {
	// Nullable: a re-parse of the exam deletes and reinserts question
	// rows, which SET NULLs this while the gap record survives.
	QuestionID          *uuid.UUID `json:"questionId"`
	SymptomTopicID      uuid.UUID  `json:"symptomTopicId"`
	SymptomTopicTitle   string     `json:"symptomTopicTitle"`
	CloCode             *string    `json:"cloCode"`
	RootCauseTopicID    *uuid.UUID `json:"rootCauseTopicId"`
	RootCauseTopicTitle *string    `json:"rootCauseTopicTitle"`
	RootCauseGrade      *int       `json:"rootCauseGrade"`
	SeverityScore       float64    `json:"severityScore"`
	PrerequisiteDepth   int        `json:"prerequisiteDepth"`
	LlmExplanation      *string    `json:"llmExplanation"`
}

// ExamInsight is the Exam Insight Layer (students.exam_insights) plus its
// granular gap records -- what a student sees on a past exam's result page.
type ExamInsight struct {
	AttemptID   uuid.UUID        `json:"attemptId"`
	ExamID      uuid.UUID        `json:"examId"`
	TotalScore  *float64         `json:"totalScore"`
	Percentage  *float64         `json:"percentage"`
	Passed      *bool            `json:"passed"`
	GapsFound   int              `json:"gapsFound"`
	Summary     *string          `json:"summary"`
	LlmModel    *string          `json:"llmModel"`
	GeneratedAt time.Time        `json:"generatedAt"`
	Gaps        []GapRecordEntry `json:"gaps"`
}

// ExamInsightListEntry backs the teacher-facing per-exam insight list --
// summary only, no gap detail (the teacher drills into a student via the
// student-facing shape if needed).
type ExamInsightListEntry struct {
	StudentID   uuid.UUID `json:"studentId"`
	AttemptID   uuid.UUID `json:"attemptId"`
	TotalScore  *float64  `json:"totalScore"`
	Percentage  *float64  `json:"percentage"`
	Passed      *bool     `json:"passed"`
	GapsFound   int       `json:"gapsFound"`
	Summary     *string   `json:"summary"`
	GeneratedAt time.Time `json:"generatedAt"`
}

// SubjectProfile is the Subject Health Layer (students.subject_profiles):
// one rolling health score per subject regardless of which exams fed it.
type SubjectProfile struct {
	SubjectCode       string          `json:"subjectCode"`
	SubjectName       string          `json:"subjectName"`
	GradeLevel        int             `json:"gradeLevel"`
	CurrentMasteryPct float64         `json:"currentMasteryPct"`
	TopWeakAreas      json.RawMessage `json:"topWeakAreas"`
	ExamsAnalyzed     int             `json:"examsAnalyzed"`
	LastUpdated       time.Time       `json:"lastUpdated"`
}
