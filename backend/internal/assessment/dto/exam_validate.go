package dto

import (
	"time"

	"github.com/google/uuid"
)

// ValidationReport is Capability 2B's 5-part report, computed by
// service.ValidateExam and stored as assessment.exams.validation_report.
type ValidationReport struct {
	GeneratedAt time.Time `json:"generatedAt"`
	Scope       string    `json:"scope"` // e.g. "unit_test: units 1, 2" or "final_exam: all units"

	CLOCoverage            CLOCoverageReport            `json:"cloCoverage"`
	BloomBalance           BloomBalanceReport           `json:"bloomBalance"`
	DifficultyDistribution DifficultyDistributionReport `json:"difficultyDistribution"`
	TopicCoverage          []TopicCoverageEntry          `json:"topicCoverage"`
	PrerequisiteWarnings   []PrerequisiteWarningEntry    `json:"prerequisiteWarnings"`
}

type CLOCoverageReport struct {
	TotalMandatoryCLOs   int      `json:"totalMandatoryClos"`
	CoveredMandatoryCLOs int      `json:"coveredMandatoryClos"`
	MissingMandatoryCLOs []string `json:"missingMandatoryClos"`
	TotalCLOs            int      `json:"totalClos"`
	CoveredCLOs          int      `json:"coveredClos"`
	Summary              string   `json:"summary"`
}

// BloomBalanceReport is computed only over questions with a matched CLO
// (curriculum.clos.bloom_level is the only source of Bloom level --
// assessment.questions has no bloom column of its own).
// UnclassifiedQuestions is surfaced explicitly rather than silently
// dropped from the percentages.
type BloomBalanceReport struct {
	Counts                  map[string]int     `json:"counts"`
	Percentages             map[string]float64 `json:"percentages"`
	UnclassifiedQuestions   int                `json:"unclassifiedQuestions"`
	HigherOrderPercent      float64            `json:"higherOrderPercent"` // apply + analyse
	MinimumHigherOrderPct   float64            `json:"minimumHigherOrderPercent"`
	MeetsMinimumHigherOrder bool               `json:"meetsMinimumHigherOrder"`
	Summary                 string             `json:"summary"`
}

// DifficultyDistributionReport's difficulty is derived from the linked
// CLO's Bloom level (assessment.questions.difficulty_level is never
// populated by 2A) via a fixed remember/understand->easy,
// apply/analyse->medium, evaluate/create->hard mapping.
type DifficultyDistributionReport struct {
	Counts                map[string]int     `json:"counts"`
	Percentages           map[string]float64 `json:"percentages"`
	UnclassifiedQuestions int                `json:"unclassifiedQuestions"`
	HardPercent           float64            `json:"hardPercent"`
	MaxHardPercentAllowed float64            `json:"maxHardPercentAllowed"`
	ExceedsMaxHard        bool               `json:"exceedsMaxHard"`
	Summary               string             `json:"summary"`
}

type TopicCoverageEntry struct {
	TopicTitle    string `json:"topicTitle"`
	UnitNumber    int    `json:"unitNumber"`
	QuestionCount int    `json:"questionCount"`
}

// PrerequisiteWarningEntry: curriculum.topic_prerequisites has no rows in
// this dataset today, so PrerequisiteWarnings will legitimately come back
// empty until prerequisite relationships are added to the curriculum --
// that's a data gap, not a bug in this check.
type PrerequisiteWarningEntry struct {
	TopicTitle        string `json:"topicTitle"`
	PrerequisiteTitle string `json:"prerequisiteTitle"`
	PrerequisiteGrade int    `json:"prerequisiteGrade"`
	IsCrossGrade      bool   `json:"isCrossGrade"`
	Message           string `json:"message"`
}

type PublishResponse struct {
	ExamID uuid.UUID `json:"examId"`
	Status string    `json:"status"`
}
