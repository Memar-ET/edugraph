package dto

import "time"

// CreateStudentRequest enrolls an existing user (created via POST /auth/register
// with role=student) as a student record tied to a school.
type CreateStudentRequest struct {
	UserID      string `json:"user_id" validate:"required,uuid"`
	SchoolID    string `json:"school_id" validate:"required,uuid"`
	AdmissionNo string `json:"admission_no" validate:"required"`
	GradeLevel  int16  `json:"grade_level" validate:"required,min=1,max=12"`
}

type UpdateStudentRequest struct {
	GradeLevel int16 `json:"grade_level" validate:"required,min=1,max=12"`
}

type StudentResponse struct {
	ID          string    `json:"id"`
	UserID      string    `json:"user_id"`
	SchoolID    string    `json:"school_id"`
	AdmissionNo string    `json:"admission_no"`
	GradeLevel  int16     `json:"grade_level"`
	CreatedAt   time.Time `json:"created_at"`
}
