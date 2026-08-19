package syncagent

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/jackc/pgx/v5/pgxpool"
)

const pushBatchSize = 100

// PushOutbox reads up to pushBatchSize unsynced rows from sync.outbox,
// posts them to the cloud, then marks them pushed. Idempotent: the cloud
// uses (device_id, entity_id, entity_type) dedup logic so a retry after a
// partial acknowledgement is safe.
func PushOutbox(ctx context.Context, localDB *pgxpool.Pool, cloud *CloudClient, deviceID, schoolID string, state *SafeState) (int, error) {
	rows, err := fetchUnpushedRows(ctx, localDB, pushBatchSize)
	if err != nil {
		return 0, fmt.Errorf("fetch outbox: %w", err)
	}
	state.SetPendingRows(len(rows))
	if len(rows) == 0 {
		return 0, nil
	}

	accepted, err := cloud.PostSyncPush(ctx, schoolID, rows)
	if err != nil {
		return 0, fmt.Errorf("cloud push: %w", err)
	}

	// Mark all rows in this batch as pushed regardless of the accepted count —
	// the cloud is idempotent so any it already had are fine too.
	ids := make([]int64, len(rows))
	for i, r := range rows {
		ids[i] = r.ID
	}
	if err := markPushed(ctx, localDB, ids); err != nil {
		slog.WarnContext(ctx, "mark pushed failed — rows will be retried next cycle", "err", err)
	}

	return accepted, nil
}

func fetchUnpushedRows(ctx context.Context, db *pgxpool.Pool, limit int) ([]OutboxRow, error) {
	const q = `
		SELECT id, entity_type, entity_id, COALESCE(school_id::text,''), operation, payload, created_at
		FROM sync.outbox
		WHERE pushed_at IS NULL
		ORDER BY created_at ASC
		LIMIT $1`
	rows, err := db.Query(ctx, q, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []OutboxRow
	for rows.Next() {
		var r OutboxRow
		if err := rows.Scan(&r.ID, &r.EntityType, &r.EntityID, &r.SchoolID, &r.Operation, &r.Payload, &r.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

func markPushed(ctx context.Context, db *pgxpool.Pool, ids []int64) error {
	const q = `UPDATE sync.outbox SET pushed_at = NOW() WHERE id = ANY($1)`
	_, err := db.Exec(ctx, q, ids)
	return err
}
