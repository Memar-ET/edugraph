package dto

import "time"

type CreateSubjectRequest struct {
	Name       string `json:"name" validate:"required"`
	Code       string `json:"code" validate:"required"`
	GradeLevel int16  `json:"grade_level" validate:"required,min=1,max=12"`
}

type SubjectResponse struct {
	ID         string    `json:"id"`
	Name       string    `json:"name"`
	Code       string    `json:"code"`
	GradeLevel int16     `json:"grade_level"`
	CreatedAt  time.Time `json:"created_at"`
}

type CreateUnitRequest struct {
	SubjectID   string `json:"subject_id" validate:"required,uuid"`
	Title       string `json:"title" validate:"required"`
	Description string `json:"description,omitempty"`
	OrderIndex  int    `json:"order_index"`
}

type UpdateUnitRequest struct {
	Title       string `json:"title" validate:"required"`
	Description string `json:"description,omitempty"`
	OrderIndex  int    `json:"order_index"`
}

type UnitResponse struct {
	ID          string    `json:"id"`
	SubjectID   string    `json:"subject_id"`
	Title       string    `json:"title"`
	Description string    `json:"description,omitempty"`
	OrderIndex  int       `json:"order_index"`
	CreatedAt   time.Time `json:"created_at"`
}

// AddPrerequisiteRequest links unit -> prerequisite in the Neo4j graph:
// (:CurriculumUnit {id: unit_id})-[:PREREQUISITE_OF]->(:CurriculumUnit {id: prerequisite of}).
// Semantically: PrerequisiteUnitID must be completed before UnitID.
type AddPrerequisiteRequest struct {
	PrerequisiteUnitID string `json:"prerequisite_unit_id" validate:"required,uuid"`
}

type PrerequisiteResponse struct {
	UnitID string `json:"unit_id"`
	Title  string `json:"title"`
}
