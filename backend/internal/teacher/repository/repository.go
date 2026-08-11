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

type Teacher struct {
	ID               string
	UserID           string
	SchoolID         string
	SubjectSpecialty *string
	CreatedAt        time.Time
	UpdatedAt        time.Time
}

type CreateTeacherParams struct {
	UserID           string
	SchoolID         string
	SubjectSpecialty *string
}

type Repository struct {
	pool *pgxpool.Pool
}

func New(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

func (r *Repository) Create(ctx context.Context, p CreateTeacherParams) (Teacher, error) {
	const q = `INSERT INTO teachers (user_id, school_id, subject_specialty) VALUES ($1, $2, $3)
		RETURNING id, user_id, school_id, subject_specialty, created_at, updated_at`
	return scanTeacher(r.pool.QueryRow(ctx, q, p.UserID, p.SchoolID, p.SubjectSpecialty))
}

func (r *Repository) GetByID(ctx context.Context, id string) (Teacher, error) {
	const q = `SELECT id, user_id, school_id, subject_specialty, created_at, updated_at
		FROM teachers WHERE id = $1`
	t, err := scanTeacher(r.pool.QueryRow(ctx, q, id))
	if errors.Is(err, pgx.ErrNoRows) {
		return Teacher{}, ErrNotFound
	}
	return t, err
}

func (r *Repository) List(ctx context.Context, schoolID string, limit, offset int) ([]Teacher, int64, error) {
	q := `SELECT id, user_id, school_id, subject_specialty, created_at, updated_at FROM teachers`
	countQ := `SELECT count(*) FROM teachers`
	args := []any{}
	if schoolID != "" {
		q += ` WHERE school_id = $1`
		countQ += ` WHERE school_id = $1`
		args = append(args, schoolID)
	}
	q += fmt.Sprintf(` ORDER BY created_at LIMIT %d OFFSET %d`, limit, offset)

	rows, err := r.pool.Query(ctx, q, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("list teachers: %w", err)
	}
	defer rows.Close()

	var teachers []Teacher
	for rows.Next() {
		t, err := scanTeacher(rows)
		if err != nil {
			return nil, 0, err
		}
		teachers = append(teachers, t)
	}

	var total int64
	if err := r.pool.QueryRow(ctx, countQ, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count teachers: %w", err)
	}

	return teachers, total, nil
}

func (r *Repository) Update(ctx context.Context, id string, specialty *string) (Teacher, error) {
	const q = `UPDATE teachers SET subject_specialty = $2, updated_at = now() WHERE id = $1
		RETURNING id, user_id, school_id, subject_specialty, created_at, updated_at`
	t, err := scanTeacher(r.pool.QueryRow(ctx, q, id, specialty))
	if errors.Is(err, pgx.ErrNoRows) {
		return Teacher{}, ErrNotFound
	}
	return t, err
}

func (r *Repository) Delete(ctx context.Context, id string) error {
	tag, err := r.pool.Exec(ctx, `DELETE FROM teachers WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("delete teacher: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func scanTeacher(row pgx.Row) (Teacher, error) {
	var t Teacher
	err := row.Scan(&t.ID, &t.UserID, &t.SchoolID, &t.SubjectSpecialty, &t.CreatedAt, &t.UpdatedAt)
	if err != nil {
		return Teacher{}, fmt.Errorf("scan teacher: %w", err)
	}
	return t, nil
}
