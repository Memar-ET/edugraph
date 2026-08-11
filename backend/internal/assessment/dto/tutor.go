package dto

// TutorAskRequest is a student's question to the AI tutor (Capability
// 3C). Language mirrors study plans: en / am, defaulting to en.
type TutorAskRequest struct {
	Question string  `json:"question" validate:"required,min=3,max=2000"`
	Language *string `json:"language" validate:"omitempty,oneof=en am"`
}
