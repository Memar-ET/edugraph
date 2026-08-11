package dto

import "time"

type CreateJobRequest struct {
	Type    string         `json:"type" validate:"required"`
	Payload map[string]any `json:"payload,omitempty"`
}

type UpdateJobStatusRequest struct {
	Status string         `json:"status" validate:"required,oneof=pending running completed failed"`
	Result map[string]any `json:"result,omitempty"`
	Error  string         `json:"error,omitempty"`
}

type JobResponse struct {
	ID        string         `json:"id"`
	Type      string         `json:"type"`
	Status    string         `json:"status"`
	Payload   map[string]any `json:"payload,omitempty"`
	Result    map[string]any `json:"result,omitempty"`
	Error     string         `json:"error,omitempty"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
}
