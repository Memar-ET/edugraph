package syncagent

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"sync"
	"time"
)

// SafeState wraps AgentState with a mutex for concurrent access from the
// push/pull goroutines and the health HTTP server.
type SafeState struct {
	mu   sync.RWMutex
	data AgentState
}

func (s *SafeState) SetLastPush(t time.Time, errMsg string) {
	s.mu.Lock()
	s.data.LastPushAt = &t
	s.data.LastPushErr = errMsg
	s.mu.Unlock()
}

func (s *SafeState) SetLastPull(t time.Time, errMsg string) {
	s.mu.Lock()
	s.data.LastPullAt = &t
	s.data.LastPullErr = errMsg
	s.mu.Unlock()
}

func (s *SafeState) SetPendingRows(n int) {
	s.mu.Lock()
	s.data.PendingRows = n
	s.mu.Unlock()
}

func (s *SafeState) Snapshot() AgentState {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.data
}

// StartHealthServer runs a minimal HTTP server on addr (e.g. ":8081")
// serving GET /health. Used by School Box monitoring scripts and docker
// healthcheck probes.
func StartHealthServer(ctx context.Context, addr string, state *SafeState) {
	mux := http.NewServeMux()
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		snap := state.Snapshot()
		w.Header().Set("Content-Type", "application/json")

		type response struct {
			Status      string     `json:"status"`
			LastPushAt  *time.Time `json:"last_push_at"`
			LastPullAt  *time.Time `json:"last_pull_at"`
			PendingRows int        `json:"pending_rows"`
			LastPushErr string     `json:"last_push_error,omitempty"`
			LastPullErr string     `json:"last_pull_error,omitempty"`
		}
		status := "ok"
		if snap.LastPushErr != "" || snap.LastPullErr != "" {
			status = "degraded"
		}
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(response{
			Status:      status,
			LastPushAt:  snap.LastPushAt,
			LastPullAt:  snap.LastPullAt,
			PendingRows: snap.PendingRows,
			LastPushErr: snap.LastPushErr,
			LastPullErr: snap.LastPullErr,
		})
	})

	srv := &http.Server{Addr: addr, Handler: mux}
	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("health server error", "err", err)
		}
	}()
	go func() {
		<-ctx.Done()
		_ = srv.Shutdown(context.Background())
	}()
}
