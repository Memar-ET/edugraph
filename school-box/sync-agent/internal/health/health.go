// Package health exposes the sync-agent's own liveness/sync status over
// HTTP so School Box operational tooling (checklist 13.3 — monitoring the
// box itself — is a separate, larger piece; this is just this process's
// slice of it) can tell "running but can't reach the cloud" apart from
// "crashed."
package health

import (
	"encoding/json"
	"net/http"
	"sync"
	"time"
)

// State is updated by the sync agent after every cycle and read by the
// HTTP handlers below. Safe for concurrent use.
type State struct {
	mu sync.RWMutex

	lastCycleAt   time.Time
	lastSuccessAt time.Time
	pendingOutbox int
	lastPushError string
	lastPullError string
}

func NewState() *State {
	return &State{}
}

func (s *State) RecordCycle(success bool, pendingOutbox int, pushErr, pullErr error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.lastCycleAt = time.Now().UTC()
	s.pendingOutbox = pendingOutbox
	if success {
		s.lastSuccessAt = s.lastCycleAt
	}
	s.lastPushError = errString(pushErr)
	s.lastPullError = errString(pullErr)
}

func errString(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}

type syncStatus struct {
	LastCycleAt         *time.Time `json:"last_cycle_at,omitempty"`
	LastSuccessAt       *time.Time `json:"last_success_at,omitempty"`
	PendingOutboxCount  int        `json:"pending_outbox_count"`
	LastPushError       string     `json:"last_push_error,omitempty"`
	LastPullError       string     `json:"last_pull_error,omitempty"`
	SecondsSinceSuccess *float64   `json:"seconds_since_success,omitempty"`
}

func (s *State) snapshot() syncStatus {
	s.mu.RLock()
	defer s.mu.RUnlock()

	status := syncStatus{
		PendingOutboxCount: s.pendingOutbox,
		LastPushError:      s.lastPushError,
		LastPullError:      s.lastPullError,
	}
	if !s.lastCycleAt.IsZero() {
		t := s.lastCycleAt
		status.LastCycleAt = &t
	}
	if !s.lastSuccessAt.IsZero() {
		t := s.lastSuccessAt
		status.LastSuccessAt = &t
		secs := time.Since(s.lastSuccessAt).Seconds()
		status.SecondsSinceSuccess = &secs
	}
	return status
}

// NewServer builds the health HTTP server. GET /health is a plain
// liveness probe (200 if the process is up); GET /health/sync reports
// actual sync status for anything that wants more than "is it running."
func NewServer(addr string, state *State) *http.Server {
	mux := http.NewServeMux()
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"ok"}`))
	})
	mux.HandleFunc("/health/sync", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(state.snapshot())
	})
	return &http.Server{Addr: addr, Handler: mux}
}
