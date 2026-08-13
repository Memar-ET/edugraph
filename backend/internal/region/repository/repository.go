package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrNotFound = errors.New("not found")

type Region struct {
	ID        string
	Name      string
	Code      string
	CreatedAt time.Time
	UpdatedAt time.Time
}

type Repository struct {
	pool *pgxpool.Pool
}

func New(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

func (r *Repository) Create(ctx context.Context, name, code string) (Region, error) {
	const q = `INSERT INTO regions (name, code) VALUES ($1, $2)
		RETURNING id, name, code, created_at, updated_at`
	return scanRegion(r.pool.QueryRow(ctx, q, name, code))
}

func (r *Repository) GetByID(ctx context.Context, id string) (Region, error) {
	const q = `SELECT id, name, code, created_at, updated_at FROM regions WHERE id = $1`
	reg, err := scanRegion(r.pool.QueryRow(ctx, q, id))
	if errors.Is(err, pgx.ErrNoRows) {
		return Region{}, ErrNotFound
	}
	return reg, err
}

func (r *Repository) List(ctx context.Context, limit, offset int) ([]Region, int64, error) {
	const q = `SELECT id, name, code, created_at, updated_at FROM regions
		ORDER BY name LIMIT $1 OFFSET $2`
	rows, err := r.pool.Query(ctx, q, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("list regions: %w", err)
	}
	defer rows.Close()

	var regions []Region
	for rows.Next() {
		reg, err := scanRegion(rows)
		if err != nil {
			return nil, 0, err
		}
		regions = append(regions, reg)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("list regions: %w", err)
	}

	var total int64
	if err := r.pool.QueryRow(ctx, `SELECT count(*) FROM regions`).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count regions: %w", err)
	}

	return regions, total, nil
}

func (r *Repository) Update(ctx context.Context, id, name string) (Region, error) {
	const q = `UPDATE regions SET name = $2, updated_at = now() WHERE id = $1
		RETURNING id, name, code, created_at, updated_at`
	reg, err := scanRegion(r.pool.QueryRow(ctx, q, id, name))
	if errors.Is(err, pgx.ErrNoRows) {
		return Region{}, ErrNotFound
	}
	return reg, err
}

func (r *Repository) Delete(ctx context.Context, id string) error {
	tag, err := r.pool.Exec(ctx, `DELETE FROM regions WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("delete region: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func scanRegion(row pgx.Row) (Region, error) {
	var reg Region
	if err := row.Scan(&reg.ID, &reg.Name, &reg.Code, &reg.CreatedAt, &reg.UpdatedAt); err != nil {
		return Region{}, fmt.Errorf("scan region: %w", err)
	}
	return reg, nil
}
