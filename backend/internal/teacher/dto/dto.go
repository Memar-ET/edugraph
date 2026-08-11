package dto

import "time"

// CreateTeacherRequest links an existing user (created via POST /auth/register
// with role=teacher) to a school as a teacher record.
type CreateTeacherRequest struct {
	UserID           string `json:"user_id" validate:"required,uuid"`
	SchoolID         string `json:"school_id" validate:"required,uuid"`
	SubjectSpecialty string `json:"subject_specialty,omitempty"`
}

type UpdateTeacherRequest struct {
	SubjectSpecialty string `json:"subject_specialty,omitempty"`
}

type TeacherResponse struct {
	ID               string    `json:"id"`
	UserID           string    `json:"user_id"`
	SchoolID         string    `json:"school_id"`
	SubjectSpecialty string    `json:"subject_specialty,omitempty"`
	CreatedAt        time.Time `json:"created_at"`
}
