package repository

import (
	"context"
	"errors"
	"fmt"
	"strings"
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
		FROM teachers WHERE id = $1 AND deleted_at IS NULL`
	t, err := scanTeacher(r.pool.QueryRow(ctx, q, id))
	if errors.Is(err, pgx.ErrNoRows) {
		return Teacher{}, ErrNotFound
	}
	return t, err
}

// CallerScope resolves the calling user's own school/region for
// server-side scoping -- see student/repository.go's identical method
// for the full rationale (never trust a client-supplied school_id).
func (r *Repository) CallerScope(ctx context.Context, userID string) (schoolID, regionID string, err error) {
	var s, rg *string
	err = r.pool.QueryRow(ctx, `SELECT school_id, region_id FROM users WHERE id = $1`, userID).Scan(&s, &rg)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", "", ErrNotFound
	}
	if err != nil {
		return "", "", fmt.Errorf("lookup caller scope: %w", err)
	}
	if s != nil {
		schoolID = *s
	}
	if rg != nil {
		regionID = *rg
	}
	return schoolID, regionID, nil
}

// SchoolRegionID looks up which region a school belongs to, for
// regional_admin's Get authorization check.
func (r *Repository) SchoolRegionID(ctx context.Context, schoolID string) (string, error) {
	var regionID string
	err := r.pool.QueryRow(ctx, `SELECT region_id FROM schools WHERE id = $1`, schoolID).Scan(&regionID)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", ErrNotFound
	}
	if err != nil {
		return "", fmt.Errorf("lookup school region: %w", err)
	}
	return regionID, nil
}

// List: schoolID and regionID are optional filters, ANDed when both are
// set -- see student/repository.go's List for the full rationale.
func (r *Repository) List(ctx context.Context, schoolID, regionID string, limit, offset int) ([]Teacher, int64, error) {
	base := `FROM teachers t`
	if regionID != "" {
		base += ` JOIN schools sc ON sc.id = t.school_id`
	}
	where := []string{"t.deleted_at IS NULL"}
	var args []any
	if schoolID != "" {
		args = append(args, schoolID)
		where = append(where, fmt.Sprintf("t.school_id = $%d", len(args)))
	}
	if regionID != "" {
		args = append(args, regionID)
		where = append(where, fmt.Sprintf("sc.region_id = $%d", len(args)))
	}
	whereClause := ""
	if len(where) > 0 {
		whereClause = " WHERE " + strings.Join(where, " AND ")
	}

	q := fmt.Sprintf(`SELECT t.id, t.user_id, t.school_id, t.subject_specialty, t.created_at, t.updated_at
		%s%s ORDER BY t.created_at LIMIT %d OFFSET %d`, base, whereClause, limit, offset)
	countQ := fmt.Sprintf(`SELECT count(*) %s%s`, base, whereClause)

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
	const q = `UPDATE teachers SET subject_specialty = $2, updated_at = now() WHERE id = $1 AND deleted_at IS NULL
		RETURNING id, user_id, school_id, subject_specialty, created_at, updated_at`
	t, err := scanTeacher(r.pool.QueryRow(ctx, q, id, specialty))
	if errors.Is(err, pgx.ErrNoRows) {
		return Teacher{}, ErrNotFound
	}
	return t, err
}

// Delete soft-deletes -- see student/repository.go's identical method
// for the full rationale.
func (r *Repository) Delete(ctx context.Context, id string) error {
	tag, err := r.pool.Exec(ctx, `UPDATE teachers SET deleted_at = now() WHERE id = $1 AND deleted_at IS NULL`, id)
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
