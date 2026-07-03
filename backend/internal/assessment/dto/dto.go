package dto

import "time"

type CreateAssessmentRequest struct {
	SubjectID  string `json:"subject_id" validate:"required,uuid"`
	SchoolID   string `json:"school_id,omitempty" validate:"omitempty,uuid"`
	Title      string `json:"title" validate:"required"`
	Type       string `json:"type" validate:"required,oneof=quiz exam national"`
	TotalMarks int    `json:"total_marks" validate:"min=0"`
}

type AssessmentResponse struct {
	ID         string    `json:"id"`
	SubjectID  string    `json:"subject_id"`
	SchoolID   string    `json:"school_id,omitempty"`
	Title      string    `json:"title"`
	Type       string    `json:"type"`
	TotalMarks int       `json:"total_marks"`
	CreatedAt  time.Time `json:"created_at"`
}

type CreateQuestionRequest struct {
	QuestionText  string   `json:"question_text" validate:"required"`
	Options       []string `json:"options,omitempty"`
	CorrectAnswer string   `json:"correct_answer" validate:"required"`
	Marks         int      `json:"marks" validate:"min=1"`
	OrderIndex    int      `json:"order_index"`
}

// QuestionResponse deliberately omits the correct answer — it's used to
// render the assessment to students/teachers, not to grade client-side.
type QuestionResponse struct {
	ID           string   `json:"id"`
	AssessmentID string   `json:"assessment_id"`
	QuestionText string   `json:"question_text"`
	Options      []string `json:"options,omitempty"`
	Marks        int      `json:"marks"`
	OrderIndex   int      `json:"order_index"`
}

type SubmitResultRequest struct {
	StudentID string            `json:"student_id" validate:"required,uuid"`
	Answers   map[string]string `json:"answers" validate:"required"`
}

type ResultResponse struct {
	ID           string     `json:"id"`
	AssessmentID string     `json:"assessment_id"`
	StudentID    string     `json:"student_id"`
	Score        float64    `json:"score"`
	SubmittedAt  time.Time  `json:"submitted_at"`
	GradedAt     *time.Time `json:"graded_at,omitempty"`
}
