package dto

import "time"

type CreateSchoolRequest struct {
	RegionID string `json:"region_id" validate:"required,uuid"`
	Name     string `json:"name" validate:"required"`
	Code     string `json:"code" validate:"required"`
	Address  string `json:"address,omitempty"`
}

type UpdateSchoolRequest struct {
	Name    string `json:"name" validate:"required"`
	Address string `json:"address,omitempty"`
}

type SchoolResponse struct {
	ID        string    `json:"id"`
	RegionID  string    `json:"region_id"`
	Name      string    `json:"name"`
	Code      string    `json:"code"`
	Address   string    `json:"address,omitempty"`
	CreatedAt time.Time `json:"created_at"`
}
