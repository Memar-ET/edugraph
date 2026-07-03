package dto

import "time"

type CreateRegionRequest struct {
	Name string `json:"name" validate:"required"`
	Code string `json:"code" validate:"required"`
}

type UpdateRegionRequest struct {
	Name string `json:"name" validate:"required"`
}

type RegionResponse struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Code      string    `json:"code"`
	CreatedAt time.Time `json:"created_at"`
}
