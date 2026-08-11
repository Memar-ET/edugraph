// Package sync implements the School Box side of offline-first sync: draining
// the local sync.outbox (see backend/db/migrations/V026__sync_outbox.sql)
// to the cloud, and applying changes pulled back down.
package sync

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// OutboxItem is one locally-changed row, staged for push to the cloud.
type OutboxItem struct {
	ID         int64
	EntityType string
	EntityID   string
	Operation  string
	Payload    map[string]any
}

// Queue owns all access to this device's local outbox and sync-state
// tables. It never touches the domain tables themselves — that's crdt.go.
type Queue struct {
	pool *pgxpool.Pool
}

func NewQueue(pool *pgxpool.Pool) *Queue {
	return &Queue{pool: pool}
}

// FetchPending returns up to limit unpushed outbox rows, oldest first, so a
// batch push preserves the order changes actually happened in.
func (q *Queue) FetchPending(ctx context.Context, limit int) ([]OutboxItem, error) {
	const query = `
		SELECT id, entity_type, entity_id, operation, payload
		FROM sync.outbox
		WHERE pushed_at IS NULL
		ORDER BY created_at
		LIMIT $1`

	rows, err := q.pool.Query(ctx, query, limit)
	if err != nil {
		return nil, fmt.Errorf("fetch pending outbox rows: %w", err)
	}
	defer rows.Close()

	var items []OutboxItem
	for rows.Next() {
		var item OutboxItem
		var rawPayload []byte
		if err := rows.Scan(&item.ID, &item.EntityType, &item.EntityID, &item.Operation, &rawPayload); err != nil {
			return nil, fmt.Errorf("scan outbox row: %w", err)
		}
		if err := json.Unmarshal(rawPayload, &item.Payload); err != nil {
			return nil, fmt.Errorf("unmarshal outbox payload: %w", err)
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

// HasPending reports whether entityType/entityID has an unpushed local
// change — used by crdt.go to detect a genuine conflict when a pulled
// change for the same entity arrives before the local one has synced.
func (q *Queue) HasPending(ctx context.Context, entityType, entityID string) (bool, error) {
	const query = `SELECT EXISTS(SELECT 1 FROM sync.outbox WHERE entity_type = $1 AND entity_id = $2 AND pushed_at IS NULL)`
	var exists bool
	if err := q.pool.QueryRow(ctx, query, entityType, entityID).Scan(&exists); err != nil {
		return false, fmt.Errorf("check pending outbox entry: %w", err)
	}
	return exists, nil
}

// MarkPushed flags rows as delivered once the cloud has accepted them.
// At-least-once delivery: if the process dies after the cloud responds but
// before this commits, the same rows get pushed again next cycle. The
// cloud's sync_logs has no idempotency key today, so exactly-once would
// need a cloud-side change — out of scope here.
func (q *Queue) MarkPushed(ctx context.Context, ids []int64) error {
	if len(ids) == 0 {
		return nil
	}
	const query = `UPDATE sync.outbox SET pushed_at = now() WHERE id = ANY($1)`
	if _, err := q.pool.Exec(ctx, query, ids); err != nil {
		return fmt.Errorf("mark outbox rows pushed: %w", err)
	}
	return nil
}

// CountPending is exposed for the health endpoint.
func (q *Queue) CountPending(ctx context.Context) (int, error) {
	const query = `SELECT count(*) FROM sync.outbox WHERE pushed_at IS NULL`
	var n int
	if err := q.pool.QueryRow(ctx, query).Scan(&n); err != nil {
		return 0, fmt.Errorf("count pending outbox rows: %w", err)
	}
	return n, nil
}

// DeviceState is this device's sync cursor and last-activity timestamps,
// persisted so a restart doesn't lose the pull cursor.
type DeviceState struct {
	LastPushedAt *time.Time
	LastPulledAt *time.Time
}

func (q *Queue) LoadState(ctx context.Context, deviceID string) (DeviceState, error) {
	const query = `SELECT last_pushed_at, last_pulled_at FROM sync.device_state WHERE device_id = $1`
	var state DeviceState
	err := q.pool.QueryRow(ctx, query, deviceID).Scan(&state.LastPushedAt, &state.LastPulledAt)
	if err == pgx.ErrNoRows {
		return DeviceState{}, nil // first run: zero-value cursor pulls everything.
	}
	if err != nil {
		return DeviceState{}, fmt.Errorf("load device state: %w", err)
	}
	return state, nil
}

func (q *Queue) SaveState(ctx context.Context, deviceID string, state DeviceState) error {
	const query = `
		INSERT INTO sync.device_state (device_id, last_pushed_at, last_pulled_at, updated_at)
		VALUES ($1, $2, $3, now())
		ON CONFLICT (device_id) DO UPDATE SET
			last_pushed_at = COALESCE(EXCLUDED.last_pushed_at, sync.device_state.last_pushed_at),
			last_pulled_at = COALESCE(EXCLUDED.last_pulled_at, sync.device_state.last_pulled_at),
			updated_at = now()`
	if _, err := q.pool.Exec(ctx, query, deviceID, state.LastPushedAt, state.LastPulledAt); err != nil {
		return fmt.Errorf("save device state: %w", err)
	}
	return nil
}
