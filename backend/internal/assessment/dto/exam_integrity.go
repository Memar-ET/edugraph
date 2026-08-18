package dto

import (
	"encoding/json"
	"time"
)

// IntegrityEventInput is one client-reported signal (tab visibility,
// fullscreen, connection status) for the caller's current/most recent
// attempt. sequenceNumber is client-assigned and monotonic per attempt --
// the unique (attempt_id, sequence_number) constraint (V057) makes a
// resent batch after a retry idempotent rather than double-counting.
type IntegrityEventInput struct {
	EventType      string          `json:"eventType" validate:"required,oneof=tab_hidden tab_visible fullscreen_entered fullscreen_exited connection_lost connection_restored"`
	OccurredAt     time.Time       `json:"occurredAt" validate:"required"`
	SequenceNumber int             `json:"sequenceNumber" validate:"required,gte=0"`
	Metadata       json.RawMessage `json:"metadata,omitempty"`
}

type ReportIntegrityEventsRequest struct {
	Events []IntegrityEventInput `json:"events" validate:"required,min=1,max=50,dive"`
}
