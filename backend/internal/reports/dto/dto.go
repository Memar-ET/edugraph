package dto

import (
	"encoding/json"
	"time"
)

type GenerateReportRequest struct {
	ReportType string          `json:"reportType" validate:"required,oneof=school_monthly national_heatmap clo_coverage"`
	Params     json.RawMessage `json:"params" validate:"required"`
}

type ReportResponse struct {
	ID          string          `json:"id"`
	ReportType  string          `json:"reportType"`
	Params      json.RawMessage `json:"params,omitempty"`
	Result      json.RawMessage `json:"result,omitempty"`
	RequesterID string          `json:"requesterId"`
	Status      string          `json:"status"`
	ErrorText   *string         `json:"errorText,omitempty"`
	GeneratedAt *time.Time      `json:"generatedAt,omitempty"`
	CreatedAt   time.Time       `json:"createdAt"`
	UpdatedAt   time.Time       `json:"updatedAt"`
}
