package repository

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type SyncLog struct {
	ID         string
	DeviceID   string
	SchoolID   string
	EntityType string
	EntityID   string
	Operation  string
	Payload    map[string]any
	Status     string
	CreatedAt  time.Time
	SyncedAt   *time.Time
}

type CreateLogParams struct {
	DeviceID   string
	SchoolID   string
	EntityType string
	EntityID   string
	Operation  string
	Payload    map[string]any
}

type Repository struct {
	pool *pgxpool.Pool
}

func New(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

func (r *Repository) CreateLog(ctx context.Context, p CreateLogParams) (SyncLog, error) {
	payloadJSON, err := json.Marshal(p.Payload)
	if err != nil {
		return SyncLog{}, fmt.Errorf("marshal payload: %w", err)
	}

	const q = `INSERT INTO sync_logs (device_id, school_id, entity_type, entity_id, operation, payload)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, device_id, school_id, entity_type, entity_id, operation, payload, status, created_at, synced_at`
	return scanLog(r.pool.QueryRow(ctx, q, p.DeviceID, p.SchoolID, p.EntityType, p.EntityID, p.Operation, payloadJSON))
}

// ListApplied returns changes applied for schoolID after `since`, for a
// device to pull down and merge into its local store.
func (r *Repository) ListApplied(ctx context.Context, schoolID string, since time.Time) ([]SyncLog, error) {
	const q = `SELECT id, device_id, school_id, entity_type, entity_id, operation, payload, status, created_at, synced_at
		FROM sync_logs
		WHERE school_id = $1 AND status = 'applied' AND synced_at > $2
		ORDER BY synced_at`
	rows, err := r.pool.Query(ctx, q, schoolID, since)
	if err != nil {
		return nil, fmt.Errorf("list applied sync logs: %w", err)
	}
	defer rows.Close()

	var logs []SyncLog
	for rows.Next() {
		l, err := scanLog(rows)
		if err != nil {
			return nil, err
		}
		logs = append(logs, l)
	}
	return logs, nil
}

// ClaimPending atomically claims up to `limit` pending logs for processing,
// so multiple sync-worker replicas never double-apply the same change.
func (r *Repository) ClaimPending(ctx context.Context, limit int) ([]SyncLog, error) {
	const q = `
		WITH claimed AS (
			SELECT id FROM sync_logs
			WHERE status = 'pending'
			ORDER BY created_at
			LIMIT $1
			FOR UPDATE SKIP LOCKED
		)
		UPDATE sync_logs SET status = 'applied', synced_at = now()
		WHERE id IN (SELECT id FROM claimed)
		RETURNING id, device_id, school_id, entity_type, entity_id, operation, payload, status, created_at, synced_at`

	rows, err := r.pool.Query(ctx, q, limit)
	if err != nil {
		return nil, fmt.Errorf("claim pending sync logs: %w", err)
	}
	defer rows.Close()

	var logs []SyncLog
	for rows.Next() {
		l, err := scanLog(rows)
		if err != nil {
			return nil, err
		}
		logs = append(logs, l)
	}
	return logs, nil
}

func scanLog(row interface {
	Scan(dest ...any) error
}) (SyncLog, error) {
	var l SyncLog
	var rawPayload []byte
	err := row.Scan(&l.ID, &l.DeviceID, &l.SchoolID, &l.EntityType, &l.EntityID, &l.Operation,
		&rawPayload, &l.Status, &l.CreatedAt, &l.SyncedAt)
	if err != nil {
		return SyncLog{}, fmt.Errorf("scan sync log: %w", err)
	}
	_ = json.Unmarshal(rawPayload, &l.Payload)
	return l, nil
}
