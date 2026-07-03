package dto

import "time"

type ChangeItem struct {
	EntityType string         `json:"entity_type" validate:"required"`
	EntityID   string         `json:"entity_id" validate:"required,uuid"`
	Operation  string         `json:"operation" validate:"required,oneof=create update delete"`
	Payload    map[string]any `json:"payload"`
}

type PushRequest struct {
	DeviceID string       `json:"device_id" validate:"required"`
	SchoolID string       `json:"school_id" validate:"required,uuid"`
	Changes  []ChangeItem `json:"changes" validate:"required,min=1,dive"`
}

type PushResponse struct {
	Accepted int `json:"accepted"`
}

type PulledChange struct {
	EntityType string         `json:"entity_type"`
	EntityID   string         `json:"entity_id"`
	Operation  string         `json:"operation"`
	Payload    map[string]any `json:"payload"`
	SyncedAt   time.Time      `json:"synced_at"`
}

type PullResponse struct {
	Changes    []PulledChange `json:"changes"`
	ServerTime time.Time      `json:"server_time"`
}
