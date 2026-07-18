package dto

// QuestionOption is one MCQ choice, split out of question_text during
// parsing (see ai-service exam_parser/extractor.py) so the frontend can
// render real labeled choice buttons instead of raw prose.
type QuestionOption struct {
	Letter string `json:"letter"`
	Text   string `json:"text"`
}

// StudentQuestion is what a student sees while taking an exam --
// deliberately has no answer_key/clo_code/clo_align_* fields, unlike
// repository.QuestionForGrading which the grading paths use internally.
type StudentQuestion struct {
	ID             string           `json:"id"`
	SequenceNumber int              `json:"sequenceNumber"`
	QuestionText   string           `json:"questionText"`
	QuestionType   string           `json:"questionType"`
	Marks          int              `json:"marks"`
	PartLabel      *string          `json:"partLabel,omitempty"`
	Options        []QuestionOption `json:"options,omitempty"`
}

// GradingQuestion is the teacher-facing counterpart -- includes
// AnswerKey (useful grading reference, not a leak to the person doing the
// grading) for the "Grade Exam" spreadsheet's column headers.
type GradingQuestion struct {
	ID             string            `json:"id"`
	SequenceNumber int               `json:"sequenceNumber"`
	QuestionText   string            `json:"questionText"`
	QuestionType   string            `json:"questionType"`
	Marks          int               `json:"marks"`
	PartLabel      *string           `json:"partLabel,omitempty"`
	AnswerKey      map[string]string `json:"answerKey,omitempty"`
	Options        []QuestionOption  `json:"options,omitempty"`
}
