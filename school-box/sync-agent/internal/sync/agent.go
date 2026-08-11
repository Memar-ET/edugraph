package sync

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/edugraph-ai/edugraph-sync-agent/internal/health"
)

// Config holds everything the agent needs to reach the cloud and identify
// this School Box. Field names match the docker-compose.yml env vars
// already declared for the sync-agent service.
type Config struct {
	CloudSyncEndpoint string // e.g. https://api.edugraph.et — base URL, no path.
	SchoolBoxID       string // this device's identity, sent as device_id.
	SchoolID          string // the school's UUID, sent as school_id.
	SyncInterval      time.Duration
	PushBatchSize     int
	HTTPTimeout       time.Duration
}

// Agent drives one push-then-pull cycle on a timer. It owns no domain
// logic itself — that lives in queue.go (local storage) and crdt.go
// (merge rules); the agent just sequences them against the network.
type Agent struct {
	cfg     Config
	queue   *Queue
	applier *Applier
	client  *http.Client
	log     *slog.Logger

	// health is updated after every cycle; the health package's HTTP
	// handlers read it.
	health *health.State
}

func NewAgent(cfg Config, pool *pgxpool.Pool, healthState *health.State, log *slog.Logger) *Agent {
	queue := NewQueue(pool)
	return &Agent{
		cfg:     cfg,
		queue:   queue,
		applier: NewApplier(pool, queue),
		client:  &http.Client{Timeout: cfg.HTTPTimeout},
		log:     log,
		health:  healthState,
	}
}

// Run blocks until ctx is cancelled, running one sync cycle immediately
// and then on cfg.SyncInterval. Errors within a cycle are logged and
// reflected in health, never fatal — a School Box with no internet today
// should keep serving its LAN and just try again next tick.
func (a *Agent) Run(ctx context.Context) {
	a.runCycle(ctx)

	ticker := time.NewTicker(a.cfg.SyncInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			a.runCycle(ctx)
		}
	}
}

func (a *Agent) runCycle(ctx context.Context) {
	pushed, pushErr := a.pushPending(ctx)
	if pushErr != nil {
		a.log.Error("push cycle failed", "error", pushErr)
	} else {
		a.log.Info("push cycle complete", "pushed", pushed)
	}

	pulled, pullErr := a.pullChanges(ctx)
	if pullErr != nil {
		a.log.Error("pull cycle failed", "error", pullErr)
	} else {
		a.log.Info("pull cycle complete", "applied", pulled)
	}

	pending, _ := a.queue.CountPending(ctx)
	a.health.RecordCycle(pushErr == nil && pullErr == nil, pending, pushErr, pullErr)
}

type changeItem struct {
	EntityType string         `json:"entity_type"`
	EntityID   string         `json:"entity_id"`
	Operation  string         `json:"operation"`
	Payload    map[string]any `json:"payload"`
}

type pushRequest struct {
	DeviceID string       `json:"device_id"`
	SchoolID string       `json:"school_id"`
	Changes  []changeItem `json:"changes"`
}

type pushResponse struct {
	Accepted int `json:"accepted"`
}

// pushPending drains up to PushBatchSize outbox rows per call. A full
// outbox drains over several ticks rather than one giant request, which
// keeps a single sync cycle bounded on a low-power School Box.
func (a *Agent) pushPending(ctx context.Context) (int, error) {
	items, err := a.queue.FetchPending(ctx, a.cfg.PushBatchSize)
	if err != nil {
		return 0, err
	}
	if len(items) == 0 {
		return 0, nil
	}

	req := pushRequest{DeviceID: a.cfg.SchoolBoxID, SchoolID: a.cfg.SchoolID}
	for _, item := range items {
		req.Changes = append(req.Changes, changeItem{
			EntityType: item.EntityType,
			EntityID:   item.EntityID,
			Operation:  item.Operation,
			Payload:    item.Payload,
		})
	}

	var resp pushResponse
	if err := a.postJSON(ctx, "/api/v1/sync/push", req, &resp); err != nil {
		return 0, fmt.Errorf("push to cloud: %w", err)
	}

	ids := make([]int64, len(items))
	for i, item := range items {
		ids[i] = item.ID
	}
	if err := a.queue.MarkPushed(ctx, ids); err != nil {
		return 0, err
	}

	now := time.Now().UTC()
	if err := a.queue.SaveState(ctx, a.cfg.SchoolBoxID, DeviceState{LastPushedAt: &now}); err != nil {
		return 0, err
	}
	return len(items), nil
}

type pullResponseChange struct {
	EntityType string         `json:"entity_type"`
	EntityID   string         `json:"entity_id"`
	Operation  string         `json:"operation"`
	Payload    map[string]any `json:"payload"`
	SyncedAt   time.Time      `json:"synced_at"`
}

type pullResponse struct {
	Changes    []pullResponseChange `json:"changes"`
	ServerTime time.Time            `json:"server_time"`
}

func (a *Agent) pullChanges(ctx context.Context) (int, error) {
	state, err := a.queue.LoadState(ctx, a.cfg.SchoolBoxID)
	if err != nil {
		return 0, err
	}
	since := time.Unix(0, 0).UTC()
	if state.LastPulledAt != nil {
		since = *state.LastPulledAt
	}

	path := fmt.Sprintf("/api/v1/sync/pull?school_id=%s&since=%s",
		url.QueryEscape(a.cfg.SchoolID), url.QueryEscape(since.Format(time.RFC3339)))

	var resp pullResponse
	if err := a.getJSON(ctx, path, &resp); err != nil {
		return 0, fmt.Errorf("pull from cloud: %w", err)
	}

	applied := 0
	for _, c := range resp.Changes {
		change := PulledChange{
			EntityType: c.EntityType,
			EntityID:   c.EntityID,
			Operation:  c.Operation,
			Payload:    c.Payload,
			SyncedAt:   c.SyncedAt,
		}
		if err := a.applier.Apply(ctx, change); err != nil {
			// One bad row shouldn't block the rest of the batch; log and
			// continue, this entity is simply retried next cycle since
			// the cursor below only advances past what we actually saw.
			a.log.Error("apply pulled change failed", "entity_type", c.EntityType, "entity_id", c.EntityID, "error", err)
			continue
		}
		applied++
	}

	pulledAt := resp.ServerTime
	if err := a.queue.SaveState(ctx, a.cfg.SchoolBoxID, DeviceState{LastPulledAt: &pulledAt}); err != nil {
		return applied, err
	}
	return applied, nil
}

func (a *Agent) postJSON(ctx context.Context, path string, body, out any) error {
	buf, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("marshal request: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, a.cfg.CloudSyncEndpoint+path, bytes.NewReader(buf))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	return a.do(req, out)
}

func (a *Agent) getJSON(ctx context.Context, path string, out any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, a.cfg.CloudSyncEndpoint+path, nil)
	if err != nil {
		return err
	}
	return a.do(req, out)
}

func (a *Agent) do(req *http.Request, out any) error {
	resp, err := a.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("read response body: %w", err)
	}
	if resp.StatusCode >= 300 {
		return fmt.Errorf("cloud returned %d: %s", resp.StatusCode, string(body))
	}
	if out == nil || len(body) == 0 {
		return nil
	}
	if err := json.Unmarshal(body, out); err != nil {
		return fmt.Errorf("decode response: %w", err)
	}
	return nil
}
