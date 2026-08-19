package syncagent

import (
	"encoding/json"
	"time"
)

// OutboxRow mirrors sync.outbox — one row the School Box needs to push to cloud.
type OutboxRow struct {
	ID         int64           `json:"id"`
	EntityType string          `json:"entity_type"`
	EntityID   string          `json:"entity_id"`
	SchoolID   string          `json:"school_id"`
	Operation  string          `json:"operation"`
	Payload    json.RawMessage `json:"payload"`
	CreatedAt  time.Time       `json:"created_at"`
}

// CloudChange is one change the cloud returns during a pull.
type CloudChange struct {
	EntityType string          `json:"entity_type"`
	EntityID   string          `json:"entity_id"`
	Operation  string          `json:"operation"`
	Payload    json.RawMessage `json:"payload"`
	SyncedAt   time.Time       `json:"synced_at"`
}

// PushRequest is the body sent to POST /api/v1/sync/push.
type PushRequest struct {
	DeviceID string       `json:"device_id"`
	SchoolID string       `json:"school_id"`
	Changes  []ChangeItem `json:"changes"`
}

type ChangeItem struct {
	EntityType string         `json:"entity_type"`
	EntityID   string         `json:"entity_id"`
	Operation  string         `json:"operation"`
	Payload    map[string]any `json:"payload"`
}

// PushResponse is returned by the cloud push endpoint.
type PushResponse struct {
	Accepted int `json:"accepted"`
}

// PullResponse is returned by the cloud pull endpoint.
type PullResponse struct {
	Changes    []CloudChange `json:"changes"`
	ServerTime time.Time     `json:"server_time"`
}

// AgentState holds shared mutable state across goroutines (read via health endpoint).
type AgentState struct {
	LastPushAt   *time.Time
	LastPullAt   *time.Time
	PendingRows  int
	LastPushErr  string
	LastPullErr  string
}
