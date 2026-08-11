package repository

import (
	"context"
	"errors"
	"fmt"
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
		FROM students WHERE id = $1`
	st, err := scanStudent(r.pool.QueryRow(ctx, q, id))
	if errors.Is(err, pgx.ErrNoRows) {
		return Student{}, ErrNotFound
	}
	return st, err
}

func (r *Repository) List(ctx context.Context, schoolID string, limit, offset int) ([]Student, int64, error) {
	q := `SELECT id, user_id, school_id, admission_no, grade_level, created_at, updated_at FROM students`
	countQ := `SELECT count(*) FROM students`
	args := []any{}
	if schoolID != "" {
		q += ` WHERE school_id = $1`
		countQ += ` WHERE school_id = $1`
		args = append(args, schoolID)
	}
	q += fmt.Sprintf(` ORDER BY admission_no LIMIT %d OFFSET %d`, limit, offset)

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

	var total int64
	if err := r.pool.QueryRow(ctx, countQ, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count students: %w", err)
	}

	return students, total, nil
}

func (r *Repository) UpdateGradeLevel(ctx context.Context, id string, gradeLevel int16) (Student, error) {
	const q = `UPDATE students SET grade_level = $2, updated_at = now() WHERE id = $1
		RETURNING id, user_id, school_id, admission_no, grade_level, created_at, updated_at`
	st, err := scanStudent(r.pool.QueryRow(ctx, q, id, gradeLevel))
	if errors.Is(err, pgx.ErrNoRows) {
		return Student{}, ErrNotFound
	}
	return st, err
}

func (r *Repository) Delete(ctx context.Context, id string) error {
	tag, err := r.pool.Exec(ctx, `DELETE FROM students WHERE id = $1`, id)
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
