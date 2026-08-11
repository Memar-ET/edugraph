package repository

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrNotFound = errors.New("not found")

type Job struct {
	ID        string
	Type      string
	Status    string
	Payload   map[string]any
	Result    map[string]any
	Error     *string
	CreatedBy *string
	CreatedAt time.Time
	UpdatedAt time.Time
}

type Repository struct {
	pool *pgxpool.Pool
}

func New(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

func (r *Repository) Create(ctx context.Context, jobType string, payload map[string]any, createdBy *string) (Job, error) {
	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		return Job{}, fmt.Errorf("marshal payload: %w", err)
	}
	const q = `INSERT INTO jobs (type, status, payload, created_by) VALUES ($1, 'pending', $2, $3)
		RETURNING id, type, status, payload, result, error, created_by, created_at, updated_at`
	return scan(r.pool.QueryRow(ctx, q, jobType, payloadJSON, createdBy))
}

func (r *Repository) GetByID(ctx context.Context, id string) (Job, error) {
	const q = `SELECT id, type, status, payload, result, error, created_by, created_at, updated_at
		FROM jobs WHERE id = $1`
	j, err := scan(r.pool.QueryRow(ctx, q, id))
	if errors.Is(err, pgx.ErrNoRows) {
		return Job{}, ErrNotFound
	}
	return j, err
}

func (r *Repository) ListByCreator(ctx context.Context, createdBy string, limit, offset int) ([]Job, int64, error) {
	const q = `SELECT id, type, status, payload, result, error, created_by, created_at, updated_at
		FROM jobs WHERE created_by = $1 ORDER BY created_at DESC LIMIT $2 OFFSET $3`
	rows, err := r.pool.Query(ctx, q, createdBy, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("list jobs: %w", err)
	}
	defer rows.Close()

	var jobs []Job
	for rows.Next() {
		j, err := scan(rows)
		if err != nil {
			return nil, 0, err
		}
		jobs = append(jobs, j)
	}

	var total int64
	if err := r.pool.QueryRow(ctx, `SELECT count(*) FROM jobs WHERE created_by = $1`, createdBy).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count jobs: %w", err)
	}

	return jobs, total, nil
}

func (r *Repository) UpdateStatus(ctx context.Context, id, status string, result map[string]any, errMsg *string) (Job, error) {
	resultJSON, err := json.Marshal(result)
	if err != nil {
		return Job{}, fmt.Errorf("marshal result: %w", err)
	}
	const q = `UPDATE jobs SET status = $2, result = $3, error = $4, updated_at = now() WHERE id = $1
		RETURNING id, type, status, payload, result, error, created_by, created_at, updated_at`
	j, err := scan(r.pool.QueryRow(ctx, q, id, status, resultJSON, errMsg))
	if errors.Is(err, pgx.ErrNoRows) {
		return Job{}, ErrNotFound
	}
	return j, err
}

func scan(row pgx.Row) (Job, error) {
	var j Job
	var rawPayload, rawResult []byte
	err := row.Scan(&j.ID, &j.Type, &j.Status, &rawPayload, &rawResult, &j.Error, &j.CreatedBy, &j.CreatedAt, &j.UpdatedAt)
	if err != nil {
		return Job{}, fmt.Errorf("scan job: %w", err)
	}
	_ = json.Unmarshal(rawPayload, &j.Payload)
	_ = json.Unmarshal(rawResult, &j.Result)
	return j, nil
}
