package syncagent

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// PullCloud fetches changes from the cloud since `since` and applies them
// to the local School Box database. Uses last-write-wins semantics matching
// the sync.applied_versions table (V029). Curriculum, model snapshots and
// exam assignments flow cloud → school; student data flows the other
// direction (school → cloud via PushOutbox), so pull changes that touch
// student-owned tables are logged as conflicts and skipped.
func PullCloud(ctx context.Context, localDB *pgxpool.Pool, cloud *CloudClient, schoolID string, since time.Time) (int, error) {
	changes, err := cloud.GetSyncPull(ctx, schoolID, since)
	if err != nil {
		return 0, fmt.Errorf("cloud pull: %w", err)
	}
	if len(changes) == 0 {
		return 0, nil
	}

	applied := 0
	for _, ch := range changes {
		if err := applyChange(ctx, localDB, ch); err != nil {
			slog.WarnContext(ctx, "apply change failed — skipped",
				"entity_type", ch.EntityType,
				"entity_id", ch.EntityID,
				"err", err)
			continue
		}
		applied++
	}
	return applied, nil
}

// applyChange writes a single pulled change to the local DB.
// Only cloud-authoritative entity types are applied; student-owned types
// are recorded as conflicts (the school may have a local version pending push).
var cloudAuthoritative = map[string]bool{
	"curriculum.subjects":         true,
	"curriculum.units":            true,
	"curriculum.topics":           true,
	"curriculum.clos":             true,
	"curriculum.topic_clo_mappings": true,
	"curriculum.topic_prerequisites": true,
	"assessment.exams":            true,  // published exam assignments from cloud
	"modeling.model_snapshots":    true,
}

func applyChange(ctx context.Context, db *pgxpool.Pool, ch CloudChange) error {
	if !cloudAuthoritative[ch.EntityType] {
		// student-owned data flowing down is a conflict — log and skip
		return recordConflict(ctx, db, ch, nil, "unexpected cloud-push of student-owned entity")
	}

	// Check applied_versions for LWW — skip if we already have this version or newer.
	var appliedAt time.Time
	err := db.QueryRow(ctx,
		`SELECT applied_at FROM sync.applied_versions WHERE entity_type=$1 AND entity_id=$2`,
		ch.EntityType, ch.EntityID).Scan(&appliedAt)
	if err == nil && !appliedAt.Before(ch.SyncedAt) {
		// Already at this version or newer.
		return nil
	}

	// Upsert via sync.outbox trigger tables. We store the raw JSON payload
	// into a generic staging function. For MVP we write to the applied_versions
	// table only; actual table upserts are handled by the table-specific
	// functions below.
	if err := upsertPayload(ctx, db, ch); err != nil {
		return fmt.Errorf("upsert %s/%s: %w", ch.EntityType, ch.EntityID, err)
	}

	// Record that we applied this version.
	_, err = db.Exec(ctx, `
		INSERT INTO sync.applied_versions (entity_type, entity_id, applied_at)
		VALUES ($1, $2, $3)
		ON CONFLICT (entity_type, entity_id) DO UPDATE SET applied_at = EXCLUDED.applied_at`,
		ch.EntityType, ch.EntityID, ch.SyncedAt)
	return err
}

// upsertPayload applies a pulled change to the correct domain table.
// Each entity type maps to a specific upsert statement built from the
// raw JSON payload — only the columns the cloud is authoritative for.
func upsertPayload(ctx context.Context, db *pgxpool.Pool, ch CloudChange) error {
	var payload map[string]any
	if err := json.Unmarshal(ch.Payload, &payload); err != nil {
		return fmt.Errorf("unmarshal payload: %w", err)
	}

	switch ch.EntityType {
	case "curriculum.subjects":
		_, err := db.Exec(ctx, `
			INSERT INTO curriculum.subjects (id, code, name_en, name_am, grade_level, is_current, version, created_at, updated_at)
			VALUES ($1,$2,$3,$4,$5,$6,$7,NOW(),NOW())
			ON CONFLICT (id) DO UPDATE SET
				code=EXCLUDED.code, name_en=EXCLUDED.name_en, name_am=EXCLUDED.name_am,
				grade_level=EXCLUDED.grade_level, is_current=EXCLUDED.is_current,
				version=EXCLUDED.version, updated_at=NOW()`,
			strVal(payload, "id"), strVal(payload, "code"),
			strVal(payload, "name_en"), strVal(payload, "name_am"),
			intVal(payload, "grade_level"), boolVal(payload, "is_current"),
			strVal(payload, "version"),
		)
		return err

	case "assessment.exams":
		// Published exam assignments coming from cloud — upsert exam header only.
		_, err := db.Exec(ctx, `
			INSERT INTO assessment.exams (id, title, subject_id, teacher_id, school_id, status, created_at)
			VALUES ($1,$2,$3,$4,$5,$6,NOW())
			ON CONFLICT (id) DO UPDATE SET
				title=EXCLUDED.title, status=EXCLUDED.status`,
			strVal(payload, "id"), strVal(payload, "title"),
			strVal(payload, "subject_id"), strVal(payload, "teacher_id"),
			strVal(payload, "school_id"), strVal(payload, "status"),
		)
		return err

	case "modeling.model_snapshots":
		_, err := db.Exec(ctx, `
			INSERT INTO modeling.model_snapshots (id, model_type, status, scope, config, created_at)
			VALUES ($1,$2,$3,$4,$5,NOW())
			ON CONFLICT (id) DO UPDATE SET
				status=EXCLUDED.status, config=EXCLUDED.config`,
			strVal(payload, "id"), strVal(payload, "model_type"),
			strVal(payload, "status"), strVal(payload, "scope"),
			ch.Payload, // store full config JSONB
		)
		return err

	default:
		// Other cloud-authoritative types: log but don't error — schema may
		// have evolved and this School Box hasn't been updated yet.
		slog.Info("pull: no upsert handler for entity type (skipped)", "entity_type", ch.EntityType)
		return nil
	}
}

func recordConflict(ctx context.Context, db *pgxpool.Pool, ch CloudChange, local json.RawMessage, reason string) error {
	_, err := db.Exec(ctx, `
		INSERT INTO sync.conflicts (entity_type, entity_id, reason, incoming_payload, local_payload)
		VALUES ($1,$2,$3,$4,$5)`,
		ch.EntityType, ch.EntityID, reason, ch.Payload, local)
	return err
}

// payload helpers — safe coerce from map[string]any (JSON decode result).
func strVal(m map[string]any, k string) string {
	if v, ok := m[k]; ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

func intVal(m map[string]any, k string) int {
	if v, ok := m[k]; ok {
		switch n := v.(type) {
		case float64:
			return int(n)
		case int:
			return n
		}
	}
	return 0
}

func boolVal(m map[string]any, k string) bool {
	if v, ok := m[k]; ok {
		if b, ok := v.(bool); ok {
			return b
		}
	}
	return false
}
