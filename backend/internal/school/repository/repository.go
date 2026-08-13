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

type School struct {
	ID        string
	RegionID  string
	Name      string
	Code      string
	Address   *string
	CreatedAt time.Time
	UpdatedAt time.Time
}

type CreateSchoolParams struct {
	RegionID string
	Name     string
	Code     string
	Address  *string
}

type Repository struct {
	pool *pgxpool.Pool
}

func New(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

func (r *Repository) Create(ctx context.Context, p CreateSchoolParams) (School, error) {
	const q = `INSERT INTO schools (region_id, name, code, address) VALUES ($1, $2, $3, $4)
		RETURNING id, region_id, name, code, address, created_at, updated_at`
	return scanSchool(r.pool.QueryRow(ctx, q, p.RegionID, p.Name, p.Code, p.Address))
}

func (r *Repository) GetByID(ctx context.Context, id string) (School, error) {
	const q = `SELECT id, region_id, name, code, address, created_at, updated_at FROM schools WHERE id = $1`
	s, err := scanSchool(r.pool.QueryRow(ctx, q, id))
	if errors.Is(err, pgx.ErrNoRows) {
		return School{}, ErrNotFound
	}
	return s, err
}

// CallerRegionID resolves the calling user's own region, for
// regional_admin's List scoping -- see service.List. Empty if the
// caller's user row has no region set (e.g. ministry_admin, who isn't
// tied to one).
func (r *Repository) CallerRegionID(ctx context.Context, userID string) (string, error) {
	var regionID *string
	err := r.pool.QueryRow(ctx, `SELECT region_id FROM users WHERE id = $1`, userID).Scan(&regionID)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", ErrNotFound
	}
	if err != nil {
		return "", fmt.Errorf("lookup caller region: %w", err)
	}
	if regionID == nil {
		return "", nil
	}
	return *regionID, nil
}

func (r *Repository) List(ctx context.Context, regionID string, limit, offset int) ([]School, int64, error) {
	q := `SELECT id, region_id, name, code, address, created_at, updated_at FROM schools`
	countQ := `SELECT count(*) FROM schools`
	args := []any{}
	if regionID != "" {
		q += ` WHERE region_id = $1`
		countQ += ` WHERE region_id = $1`
		args = append(args, regionID)
	}
	q += fmt.Sprintf(` ORDER BY name LIMIT %d OFFSET %d`, limit, offset)

	rows, err := r.pool.Query(ctx, q, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("list schools: %w", err)
	}
	defer rows.Close()

	var schools []School
	for rows.Next() {
		s, err := scanSchool(rows)
		if err != nil {
			return nil, 0, err
		}
		schools = append(schools, s)
	}

	var total int64
	if err := r.pool.QueryRow(ctx, countQ, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count schools: %w", err)
	}

	return schools, total, nil
}

func (r *Repository) Update(ctx context.Context, id, name string, address *string) (School, error) {
	const q = `UPDATE schools SET name = $2, address = $3, updated_at = now() WHERE id = $1
		RETURNING id, region_id, name, code, address, created_at, updated_at`
	s, err := scanSchool(r.pool.QueryRow(ctx, q, id, name, address))
	if errors.Is(err, pgx.ErrNoRows) {
		return School{}, ErrNotFound
	}
	return s, err
}

func (r *Repository) Delete(ctx context.Context, id string) error {
	tag, err := r.pool.Exec(ctx, `DELETE FROM schools WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("delete school: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func scanSchool(row pgx.Row) (School, error) {
	var s School
	if err := row.Scan(&s.ID, &s.RegionID, &s.Name, &s.Code, &s.Address, &s.CreatedAt, &s.UpdatedAt); err != nil {
		return School{}, fmt.Errorf("scan school: %w", err)
	}
	return s, nil
}
