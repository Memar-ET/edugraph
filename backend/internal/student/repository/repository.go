package repository

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	neo4jdriver "github.com/neo4j/neo4j-go-driver/v5/neo4j"
)

var ErrNotFound = errors.New("not found")

type Student struct {
	ID          string
	UserID      string
	SchoolID    string
	AdmissionNo string
	GradeLevel  int16
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

type CreateStudentParams struct {
	UserID      string
	SchoolID    string
	AdmissionNo string
	GradeLevel  int16
}

type Repository struct {
	pool  *pgxpool.Pool
	neo4j neo4jdriver.DriverWithContext
}

func New(pool *pgxpool.Pool, neo4j neo4jdriver.DriverWithContext) *Repository {
	return &Repository{pool: pool, neo4j: neo4j}
}

func (r *Repository) Create(ctx context.Context, p CreateStudentParams) (Student, error) {
	const q = `INSERT INTO students (user_id, school_id, admission_no, grade_level)
		VALUES ($1, $2, $3, $4)
		RETURNING id, user_id, school_id, admission_no, grade_level, created_at, updated_at`
	st, err := scanStudent(r.pool.QueryRow(ctx, q, p.UserID, p.SchoolID, p.AdmissionNo, p.GradeLevel))
	if err != nil {
		return Student{}, err
	}

	// Mirror the student as a graph node so career/gap-analysis traversal
	// (student -> career/curriculum) doesn't need a Postgres round-trip.
	session := r.neo4j.NewSession(ctx, neo4jdriver.SessionConfig{AccessMode: neo4jdriver.AccessModeWrite})
	defer session.Close(ctx)
	_, err = session.Run(ctx,
		`MERGE (s:Student {id: $id}) SET s.gradeLevel = $gradeLevel`,
		map[string]any{"id": st.ID, "gradeLevel": int64(st.GradeLevel)},
	)
	if err != nil {
		return Student{}, fmt.Errorf("mirror student to neo4j: %w", err)
	}

	return st, nil
}

func (r *Repository) GetByID(ctx context.Context, id string) (Student, error) {
	const q = `SELECT id, user_id, school_id, admission_no, grade_level, created_at, updated_at
		FROM students WHERE id = $1 AND deleted_at IS NULL`
	st, err := scanStudent(r.pool.QueryRow(ctx, q, id))
	if errors.Is(err, pgx.ErrNoRows) {
		return Student{}, ErrNotFound
	}
	return st, err
}

// CallerScope resolves the calling user's own school/region for
// server-side scoping (never trust a client-supplied school_id/region_id
// for this -- see service.List/Get). Either may come back empty: a
// ministry_admin has neither set.
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
// regional_admin's Get authorization check (a school_id alone doesn't
// say whether it's in the caller's region).
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

// List: schoolID and regionID are both optional filters, ANDed together
// when both are set. Callers pass their own server-resolved scope here
// (see service.List) -- a regional_admin narrowing by schoolID can never
// see past their own region this way, since a school outside it simply
// matches zero rows rather than being trusted on its own.
func (r *Repository) List(ctx context.Context, schoolID, regionID string, limit, offset int) ([]Student, int64, error) {
	base := `FROM students s`
	if regionID != "" {
		base += ` JOIN schools sc ON sc.id = s.school_id`
	}
	where := []string{"s.deleted_at IS NULL"}
	var args []any
	if schoolID != "" {
		args = append(args, schoolID)
		where = append(where, fmt.Sprintf("s.school_id = $%d", len(args)))
	}
	if regionID != "" {
		args = append(args, regionID)
		where = append(where, fmt.Sprintf("sc.region_id = $%d", len(args)))
	}
	whereClause := ""
	if len(where) > 0 {
		whereClause = " WHERE " + strings.Join(where, " AND ")
	}

	q := fmt.Sprintf(`SELECT s.id, s.user_id, s.school_id, s.admission_no, s.grade_level, s.created_at, s.updated_at
		%s%s ORDER BY s.admission_no LIMIT %d OFFSET %d`, base, whereClause, limit, offset)
	countQ := fmt.Sprintf(`SELECT count(*) %s%s`, base, whereClause)

	rows, err := r.pool.Query(ctx, q, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("list students: %w", err)
	}
	defer rows.Close()

	var students []Student
	for rows.Next() {
		st, err := scanStudent(rows)
		if err != nil {
			return nil, 0, err
		}
		students = append(students, st)
	}
	// Without this, an error mid-stream (a malformed query, a dropped
	// connection) makes rows.Next() just return false as if the result
	// set had ended -- silently reporting "zero results" instead of the
	// real failure. Found via a test that hit exactly this (a bad
	// LIMIT/OFFSET), which returned an empty list with a nil error
	// instead of the actual Postgres error.
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("list students: %w", err)
	}

	var total int64
	if err := r.pool.QueryRow(ctx, countQ, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count students: %w", err)
	}

	return students, total, nil
}

func (r *Repository) UpdateGradeLevel(ctx context.Context, id string, gradeLevel int16) (Student, error) {
	const q = `UPDATE students SET grade_level = $2, updated_at = now() WHERE id = $1 AND deleted_at IS NULL
		RETURNING id, user_id, school_id, admission_no, grade_level, created_at, updated_at`
	st, err := scanStudent(r.pool.QueryRow(ctx, q, id, gradeLevel))
	if errors.Is(err, pgx.ErrNoRows) {
		return Student{}, ErrNotFound
	}
	return st, err
}

// Delete soft-deletes (see V031__soft_delete_and_audit_trail.sql) -- a
// hard DELETE here cascades through gap_records/exam_attempts/
// student_answers/study_plans/career_matches, permanently destroying a
// student's academic history on what's often a routine action
// (transfer, withdrawal, or a mistake).
func (r *Repository) Delete(ctx context.Context, id string) error {
	tag, err := r.pool.Exec(ctx, `UPDATE students SET deleted_at = now() WHERE id = $1 AND deleted_at IS NULL`, id)
	if err != nil {
		return fmt.Errorf("delete student: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func scanStudent(row pgx.Row) (Student, error) {
	var st Student
	err := row.Scan(&st.ID, &st.UserID, &st.SchoolID, &st.AdmissionNo, &st.GradeLevel, &st.CreatedAt, &st.UpdatedAt)
	if err != nil {
		return Student{}, fmt.Errorf("scan student: %w", err)
	}
	return st, nil
}
