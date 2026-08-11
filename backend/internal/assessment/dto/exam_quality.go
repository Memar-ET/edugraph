package dto

import "time"

// Capability 4B: Exam Quality Scoring -- retroactively grades the exam
// itself, not the students.

// QuestionQuality is the per-question breakdown behind every 4B metric.
type QuestionQuality struct {
	QuestionID     string `json:"questionId"`
	SequenceNumber int    `json:"sequenceNumber"`
	QuestionType   string `json:"questionType"`
	CLOCode        string `json:"cloCode,omitempty"`
	GradedAnswers  int    `json:"gradedAnswers"`

	// Discrimination: do students who mastered the topic actually pass?
	// -1..1; nil when no split produced two non-empty groups. Method is
	// "mastery" (mastery_records split) or "score_split" (top vs bottom
	// half by attempt percentage -- the classic index, used when mastery
	// evidence is missing).
	DiscriminationIndex  *float64 `json:"discriminationIndex,omitempty"`
	DiscriminationMethod string   `json:"discriminationMethod,omitempty"`

	// Difficulty calibration: stated (teacher/parser) vs actual
	// (performance-derived). Calibration is "unrated" (nothing stated),
	// "accurate", or "mismatch". ActualDifficulty is written back to
	// questions.calibrated_difficulty for future exams.
	StatedDifficulty string  `json:"statedDifficulty,omitempty"`
	ActualDifficulty string  `json:"actualDifficulty,omitempty"`
	AvgScoreRatio    float64 `json:"avgScoreRatio"`
	Calibration      string  `json:"calibration"`

	// Time anomaly: avg reported seconds; flagged when suspiciously low
	// vs the exam's median (copying / guessing / trivial question).
	AvgTimeSecs *float64 `json:"avgTimeSecs,omitempty"`
	TimeAnomaly bool     `json:"timeAnomaly"`
}

// CLOCoverageQuality: mandatory-CLO outcomes after the exam.
type CLOCoverageQuality struct {
	MandatoryTotal  int `json:"mandatoryTotal"`
	MandatoryTested int `json:"mandatoryTested"`
	// Tested but zero students answered any of its questions correctly
	// -- reteach immediately regardless of the class average.
	ZeroCorrect []CLOFlag `json:"zeroCorrect"`
	// Mandatory CLOs this exam never touched.
	Untested []CLOFlag `json:"untested"`
}

type CLOFlag struct {
	CLOCode     string `json:"cloCode"`
	Description string `json:"description,omitempty"`
}

type TimeAnalysisQuality struct {
	QuestionsWithTiming int      `json:"questionsWithTiming"`
	MedianAvgSecs       *float64 `json:"medianAvgSecs,omitempty"`
	// Insufficient timing data disables the metric rather than guessing.
	Evaluated bool `json:"evaluated"`
}

type ExamQualityResponse struct {
	ExamID           string `json:"examId"`
	Title            string `json:"title"`
	AttemptsAnalyzed int    `json:"attemptsAnalyzed"`
	// Mean discrimination across scorable questions, -1..1.
	DiscriminationAvg *float64 `json:"discriminationAvg,omitempty"`
	// 0..100: 0.5*discrimination + 0.2*calibration accuracy + 0.3*mandatory-CLO coverage.
	QualityScore float64             `json:"qualityScore"`
	Questions    []QuestionQuality   `json:"questions"`
	CLOCoverage  CLOCoverageQuality  `json:"cloCoverage"`
	TimeAnalysis TimeAnalysisQuality `json:"timeAnalysis"`
	ComputedAt   time.Time           `json:"computedAt"`
}
