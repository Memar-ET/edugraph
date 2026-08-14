package sync

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// applyMode mirrors the two merge rules from the platform's original CRDT
// design (edugraph-architecture.docx 7.5): last-write-wins on scalar
// fields for mutable entities, set-union (never overwrite, dedupe by
// primary key) for append-only ones.
type applyMode int

const (
	modeLastWriteWins applyMode = iota
	modeAppendOnly
)

type entitySpec struct {
	mode applyMode
}

// entityAllowlist is deliberately explicit rather than trusting whatever
// entity_type/columns arrive over the wire. The cloud's /sync/pull now
// requires a per-device credential (checklist 10.1, see
// backend/internal/sync/handler/device_auth.go and
// docs/architecture/data-integrity.md) -- this allowlist was originally
// this boundary's only defense against a compromised or buggy pull
// response writing to an arbitrary table, and stays as defense in depth
// even with device auth in place. Keep in sync with
// V029__sync_outbox.sql's triggers.
var entityAllowlist = map[string]entitySpec{
	"assessment.exams":           {mode: modeLastWriteWins},
	"assessment.questions":       {mode: modeLastWriteWins},
	"students.gap_records":       {mode: modeLastWriteWins},
	"students.mastery_records":   {mode: modeLastWriteWins},
	"students.study_plans":       {mode: modeLastWriteWins},
	"students.exam_insights":     {mode: modeLastWriteWins},
	"assessment.exam_attempts":   {mode: modeAppendOnly},
	"assessment.student_answers": {mode: modeAppendOnly},
}

// PulledChange is a change received from the cloud's GET /sync/pull.
type PulledChange struct {
	EntityType string
	EntityID   string
	Operation  string
	Payload    map[string]any
	SyncedAt   time.Time
}

// Applier applies pulled changes to the local domain tables and records
// conflicts for human review. It never fabricates data outside what the
// cloud sent — it only decides whether/how to write it locally.
type Applier struct {
	pool  *pgxpool.Pool
	queue *Queue
}

func NewApplier(pool *pgxpool.Pool, queue *Queue) *Applier {
	return &Applier{pool: pool, queue: queue}
}

// Apply merges one pulled change into the local database. Unknown
// entity_types are rejected rather than silently written.
func (a *Applier) Apply(ctx context.Context, change PulledChange) error {
	spec, known := entityAllowlist[change.EntityType]
	if !known {
		return fmt.Errorf("apply pulled change: entity type %q is not in the allowlist", change.EntityType)
	}

	if change.Operation == "delete" {
		return a.applyDelete(ctx, change)
	}

	if spec.mode == modeAppendOnly {
		return a.applyAppendOnly(ctx, change)
	}
	return a.applyLastWriteWins(ctx, change)
}

// applyAppendOnly inserts once and never overwrites — a submitted exam
// attempt or answer is an immutable event, so a duplicate pull (or a
// concurrent write from another device) is just a no-op, not a conflict.
func (a *Applier) applyAppendOnly(ctx context.Context, change PulledChange) error {
	query, args, err := buildUpsert(change.EntityType, change.Payload, false)
	if err != nil {
		return err
	}
	if _, err := a.pool.Exec(ctx, query, args...); err != nil {
		return fmt.Errorf("apply append-only change for %s: %w", change.EntityType, err)
	}
	return nil
}

// applyLastWriteWins skips changes older than what's already been applied,
// and logs (but still applies) any change that raced a not-yet-pushed
// local edit, per the platform's original "LWW auto-resolves, log for
// review" design.
func (a *Applier) applyLastWriteWins(ctx context.Context, change PulledChange) error {
	applied, err := a.lastAppliedAt(ctx, change.EntityType, change.EntityID)
	if err != nil {
		return err
	}
	if applied != nil && !change.SyncedAt.After(*applied) {
		return nil // stale duplicate; nothing to do.
	}

	if pending, err := a.queue.HasPending(ctx, change.EntityType, change.EntityID); err != nil {
		return err
	} else if pending {
		if err := a.logConflict(ctx, change); err != nil {
			return err
		}
	}

	query, args, err := buildUpsert(change.EntityType, change.Payload, true)
	if err != nil {
		return err
	}
	if _, err := a.pool.Exec(ctx, query, args...); err != nil {
		return fmt.Errorf("apply last-write-wins change for %s: %w", change.EntityType, err)
	}

	const versionQuery = `
		INSERT INTO sync.applied_versions (entity_type, entity_id, applied_at)
		VALUES ($1, $2, $3)
		ON CONFLICT (entity_type, entity_id) DO UPDATE SET applied_at = EXCLUDED.applied_at`
	if _, err := a.pool.Exec(ctx, versionQuery, change.EntityType, change.EntityID, change.SyncedAt); err != nil {
		return fmt.Errorf("record applied version for %s: %w", change.EntityType, err)
	}
	return nil
}

func (a *Applier) applyDelete(ctx context.Context, change PulledChange) error {
	query := fmt.Sprintf(`DELETE FROM %s WHERE id = $1`, change.EntityType)
	if _, err := a.pool.Exec(ctx, query, change.EntityID); err != nil {
		return fmt.Errorf("apply delete for %s: %w", change.EntityType, err)
	}
	return nil
}

func (a *Applier) lastAppliedAt(ctx context.Context, entityType, entityID string) (*time.Time, error) {
	const query = `SELECT applied_at FROM sync.applied_versions WHERE entity_type = $1 AND entity_id = $2`
	var applied time.Time
	err := a.pool.QueryRow(ctx, query, entityType, entityID).Scan(&applied)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("load applied version: %w", err)
	}
	return &applied, nil
}

func (a *Applier) logConflict(ctx context.Context, change PulledChange) error {
	incoming, err := json.Marshal(change.Payload)
	if err != nil {
		return fmt.Errorf("marshal conflict payload: %w", err)
	}
	const query = `
		INSERT INTO sync.conflicts (entity_type, entity_id, reason, incoming_payload)
		VALUES ($1, $2, $3, $4)`
	reason := "local change was still pending push when a remote change for the same entity was pulled; remote applied (last-write-wins)"
	if _, err := a.pool.Exec(ctx, query, change.EntityType, change.EntityID, reason, incoming); err != nil {
		return fmt.Errorf("log sync conflict: %w", err)
	}
	return nil
}

// buildUpsert turns a JSON row snapshot back into an INSERT .. ON CONFLICT
// statement. Column names come from the payload's own keys, which
// originated from to_jsonb() of the same allowlisted table on the sending
// side (see V029's trigger) — not from unvalidated request input.
func buildUpsert(table string, payload map[string]any, overwriteOnConflict bool) (string, []any, error) {
	if _, ok := payload["id"]; !ok {
		return "", nil, fmt.Errorf("payload for %s is missing its id column", table)
	}

	cols := make([]string, 0, len(payload))
	for col := range payload {
		cols = append(cols, col)
	}
	sort.Strings(cols) // deterministic SQL text, easier to debug/log.

	placeholders := make([]string, len(cols))
	args := make([]any, len(cols))
	for i, col := range cols {
		placeholders[i] = fmt.Sprintf("$%d", i+1)
		v, err := normalizeColumnValue(table, col, payload[col])
		if err != nil {
			return "", nil, fmt.Errorf("column %s.%s: %w", table, col, err)
		}
		args[i] = v
	}

	conflictAction := "DO NOTHING"
	if overwriteOnConflict {
		sets := make([]string, 0, len(cols))
		for _, col := range cols {
			if col == "id" {
				continue
			}
			sets = append(sets, fmt.Sprintf("%s = EXCLUDED.%s", col, col))
		}
		conflictAction = "DO UPDATE SET " + strings.Join(sets, ", ")
	}

	query := fmt.Sprintf(
		"INSERT INTO %s (%s) VALUES (%s) ON CONFLICT (id) %s",
		table, strings.Join(cols, ", "), strings.Join(placeholders, ", "), conflictAction,
	)
	return query, args, nil
}

// nativeArrayColumns lists columns whose Postgres type is a native array
// (e.g. integer[]) rather than JSONB. to_jsonb() on the sending side makes
// both look identical on the wire (a plain JSON array), so this is the one
// place that distinction has to be made explicit instead of inferred —
// everything else in the allowlisted tables round-trips as a scalar or a
// genuine JSONB column, both of which normalizeJSONValue already handles.
var nativeArrayColumns = map[string]map[string]bool{
	"assessment.exams": {"unit_numbers": true},
}

func normalizeColumnValue(table, col string, v any) (any, error) {
	if nativeArrayColumns[table][col] {
		return toInt32Array(v)
	}
	return normalizeJSONValue(v), nil
}

// toInt32Array converts a JSON-decoded array (elements are float64 per
// encoding/json's default number handling) into a native Go []int32 so
// pgx binds it as a Postgres integer[] instead of raw JSON bytes.
func toInt32Array(v any) ([]int32, error) {
	if v == nil {
		return nil, nil
	}
	raw, ok := v.([]any)
	if !ok {
		return nil, fmt.Errorf("expected a JSON array, got %T", v)
	}
	out := make([]int32, len(raw))
	for i, item := range raw {
		n, ok := item.(float64)
		if !ok {
			return nil, fmt.Errorf("expected a numeric array element, got %T", item)
		}
		out[i] = int32(n)
	}
	return out, nil
}

// normalizeJSONValue undoes the flattening json.Unmarshal does to
// map[string]any (e.g. nested objects/arrays decode as generic
// map/slice, which pgx can't bind directly) by re-marshaling anything
// that isn't already a driver-friendly scalar back to JSON bytes for a
// JSONB column.
func normalizeJSONValue(v any) any {
	switch v.(type) {
	case map[string]any, []any:
		b, err := json.Marshal(v)
		if err != nil {
			return nil
		}
		return b
	default:
		return v
	}
}
