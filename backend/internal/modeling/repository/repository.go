// Package repository backs the EG-GCKT model-governance review queue
// (Milestone 9): listing candidate modeling.model_snapshots rows the
// nightly refit_worker.py has produced, and promoting/rejecting them.
package repository

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/edugraph-ai/edugraph/internal/modeling/dto"
)

var ErrSnapshotNotFound = errors.New("model snapshot not found")

type Repository struct {
	pool *pgxpool.Pool
}

func New(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

func scanSnapshot(row pgx.Row) (*dto.ModelSnapshot, error) {
	var m dto.ModelSnapshot
	var id uuid.UUID
	var rawConfig, rawSummary []byte
	if err := row.Scan(&id, &m.ModelType, &m.Version, &m.Status, &m.Scope, &rawConfig, &rawSummary, &m.Notes, &m.CreatedAt); err != nil {
		return nil, err
	}
	m.ID = id.String()
	if rawConfig != nil {
		_ = json.Unmarshal(rawConfig, &m.Config)
	}
	if rawSummary != nil {
		_ = json.Unmarshal(rawSummary, &m.TrainingSummary)
	}
	return &m, nil
}

// ListCandidates returns every 'candidate' snapshot (across all engines),
// newest first -- the governance review queue.
func (r *Repository) ListCandidates(ctx context.Context) ([]dto.ModelSnapshot, error) {
	const q = `
		SELECT id, model_type, version, status, scope, config, training_summary, notes, created_at
		FROM modeling.model_snapshots
		WHERE status = 'candidate'
		ORDER BY created_at DESC
	`
	rows, err := r.pool.Query(ctx, q)
	if err != nil {
		return nil, fmt.Errorf("list candidate model snapshots: %w", err)
	}
	defer rows.Close()

	out := make([]dto.ModelSnapshot, 0)
	for rows.Next() {
		m, err := scanSnapshot(rows)
		if err != nil {
			return nil, fmt.Errorf("scan model snapshot: %w", err)
		}
		out = append(out, *m)
	}
	return out, rows.Err()
}

// PromoteSnapshot flips a candidate to 'active'. Exactly one snapshot may
// be 'active' per (model_type, scope) at a time -- the invariant every
// engine's get_active_model_snapshot query relies on -- so any
// currently-active snapshot of the same (model_type, scope) is flipped to
// 'superseded' and linked via superseded_by in the same transaction,
// never left dangling as a second "active" row.
func (r *Repository) PromoteSnapshot(ctx context.Context, id, reviewerID uuid.UUID) (*dto.ModelSnapshot, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin promote tx: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var modelType string
	var scope *string
	err = tx.QueryRow(ctx,
		`SELECT model_type, scope FROM modeling.model_snapshots WHERE id = $1 AND status = 'candidate'`, id,
	).Scan(&modelType, &scope)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrSnapshotNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("lookup candidate snapshot: %w", err)
	}

	var prevID *uuid.UUID
	err = tx.QueryRow(ctx, `
		UPDATE modeling.model_snapshots SET status = 'superseded'
		WHERE model_type = $1 AND scope IS NOT DISTINCT FROM $2 AND status = 'active'
		RETURNING id
	`, modelType, scope).Scan(&prevID)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return nil, fmt.Errorf("supersede previous active snapshot: %w", err)
	}
	if prevID != nil {
		if _, err := tx.Exec(ctx,
			`UPDATE modeling.model_snapshots SET superseded_by = $1 WHERE id = $2`, id, *prevID,
		); err != nil {
			return nil, fmt.Errorf("link superseded snapshot: %w", err)
		}
	}

	row := tx.QueryRow(ctx, `
		UPDATE modeling.model_snapshots
		SET status = 'active', validated_by = $2, validated_at = now()
		WHERE id = $1
		RETURNING id, model_type, version, status, scope, config, training_summary, notes, created_at
	`, id, reviewerID)
	m, err := scanSnapshot(row)
	if err != nil {
		return nil, fmt.Errorf("promote snapshot: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit promote: %w", err)
	}
	return m, nil
}

// RejectSnapshot flips a candidate to 'rejected' -- no cascading effects,
// since a rejected snapshot was never active in the first place.
func (r *Repository) RejectSnapshot(ctx context.Context, id, reviewerID uuid.UUID) (*dto.ModelSnapshot, error) {
	row := r.pool.QueryRow(ctx, `
		UPDATE modeling.model_snapshots
		SET status = 'rejected', validated_by = $2, validated_at = now()
		WHERE id = $1 AND status = 'candidate'
		RETURNING id, model_type, version, status, scope, config, training_summary, notes, created_at
	`, id, reviewerID)
	m, err := scanSnapshot(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrSnapshotNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("reject snapshot: %w", err)
	}
	return m, nil
}
