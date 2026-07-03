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

type Subject struct {
	ID         string
	Name       string
	Code       string
	GradeLevel int16
	CreatedAt  time.Time
}

type Unit struct {
	ID          string
	SubjectID   string
	Title       string
	Description *string
	OrderIndex  int
	CreatedAt   time.Time
}

type CreateUnitParams struct {
	SubjectID   string
	Title       string
	Description *string
	OrderIndex  int
}

type Repository struct {
	pool  *pgxpool.Pool
	neo4j neo4jdriver.DriverWithContext
}

func New(pool *pgxpool.Pool, neo4j neo4jdriver.DriverWithContext) *Repository {
	return &Repository{pool: pool, neo4j: neo4j}
}

// ── Subjects ─────────────────────────────────────────────────

func (r *Repository) CreateSubject(ctx context.Context, name, code string, gradeLevel int16) (Subject, error) {
	const q = `INSERT INTO subjects (name, code, grade_level) VALUES ($1, $2, $3)
		RETURNING id, name, code, grade_level, created_at`
	var s Subject
	err := r.pool.QueryRow(ctx, q, name, code, gradeLevel).Scan(&s.ID, &s.Name, &s.Code, &s.GradeLevel, &s.CreatedAt)
	if err != nil {
		return Subject{}, fmt.Errorf("create subject: %w", err)
	}

	session := r.neo4j.NewSession(ctx, neo4jdriver.SessionConfig{AccessMode: neo4jdriver.AccessModeWrite})
	defer session.Close(ctx)
	if _, err := session.Run(ctx, `MERGE (s:Subject {id: $id}) SET s.name = $name`,
		map[string]any{"id": s.ID, "name": s.Name}); err != nil {
		return Subject{}, fmt.Errorf("mirror subject to neo4j: %w", err)
	}

	return s, nil
}

func (r *Repository) ListSubjects(ctx context.Context, gradeLevel int16) ([]Subject, error) {
	q := `SELECT id, name, code, grade_level, created_at FROM subjects`
	args := []any{}
	if gradeLevel > 0 {
		q += ` WHERE grade_level = $1`
		args = append(args, gradeLevel)
	}
	q += ` ORDER BY grade_level, name`

	rows, err := r.pool.Query(ctx, q, args...)
	if err != nil {
		return nil, fmt.Errorf("list subjects: %w", err)
	}
	defer rows.Close()

	var subjects []Subject
	for rows.Next() {
		var s Subject
		if err := rows.Scan(&s.ID, &s.Name, &s.Code, &s.GradeLevel, &s.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan subject: %w", err)
		}
		subjects = append(subjects, s)
	}
	return subjects, nil
}

// ── Curriculum units ─────────────────────────────────────────

func (r *Repository) CreateUnit(ctx context.Context, p CreateUnitParams) (Unit, error) {
	const q = `INSERT INTO curriculum_units (subject_id, title, description, order_index)
		VALUES ($1, $2, $3, $4)
		RETURNING id, subject_id, title, description, order_index, created_at`
	u, err := scanUnit(r.pool.QueryRow(ctx, q, p.SubjectID, p.Title, p.Description, p.OrderIndex))
	if err != nil {
		return Unit{}, err
	}

	session := r.neo4j.NewSession(ctx, neo4jdriver.SessionConfig{AccessMode: neo4jdriver.AccessModeWrite})
	defer session.Close(ctx)
	_, err = session.Run(ctx, `
		MERGE (u:CurriculumUnit {id: $id})
		SET u.title = $title, u.subjectId = $subjectId
		WITH u
		MATCH (s:Subject {id: $subjectId})
		MERGE (s)-[:HAS_UNIT]->(u)`,
		map[string]any{"id": u.ID, "title": u.Title, "subjectId": u.SubjectID},
	)
	if err != nil {
		return Unit{}, fmt.Errorf("mirror unit to neo4j: %w", err)
	}

	return u, nil
}

func (r *Repository) GetUnit(ctx context.Context, id string) (Unit, error) {
	const q = `SELECT id, subject_id, title, description, order_index, created_at
		FROM curriculum_units WHERE id = $1`
	u, err := scanUnit(r.pool.QueryRow(ctx, q, id))
	if errors.Is(err, pgx.ErrNoRows) {
		return Unit{}, ErrNotFound
	}
	return u, err
}

func (r *Repository) ListUnits(ctx context.Context, subjectID string) ([]Unit, error) {
	const q = `SELECT id, subject_id, title, description, order_index, created_at
		FROM curriculum_units WHERE subject_id = $1 ORDER BY order_index`
	rows, err := r.pool.Query(ctx, q, subjectID)
	if err != nil {
		return nil, fmt.Errorf("list units: %w", err)
	}
	defer rows.Close()

	var units []Unit
	for rows.Next() {
		u, err := scanUnit(rows)
		if err != nil {
			return nil, err
		}
		units = append(units, u)
	}
	return units, nil
}

func (r *Repository) UpdateUnit(ctx context.Context, id, title string, description *string, orderIndex int) (Unit, error) {
	const q = `UPDATE curriculum_units SET title = $2, description = $3, order_index = $4, updated_at = now()
		WHERE id = $1
		RETURNING id, subject_id, title, description, order_index, created_at`
	u, err := scanUnit(r.pool.QueryRow(ctx, q, id, title, description, orderIndex))
	if errors.Is(err, pgx.ErrNoRows) {
		return Unit{}, ErrNotFound
	}
	if err != nil {
		return Unit{}, err
	}

	session := r.neo4j.NewSession(ctx, neo4jdriver.SessionConfig{AccessMode: neo4jdriver.AccessModeWrite})
	defer session.Close(ctx)
	if _, err := session.Run(ctx, `MATCH (u:CurriculumUnit {id: $id}) SET u.title = $title`,
		map[string]any{"id": u.ID, "title": u.Title}); err != nil {
		return Unit{}, fmt.Errorf("mirror unit update to neo4j: %w", err)
	}

	return u, nil
}

func (r *Repository) DeleteUnit(ctx context.Context, id string) error {
	tag, err := r.pool.Exec(ctx, `DELETE FROM curriculum_units WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("delete unit: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}

	session := r.neo4j.NewSession(ctx, neo4jdriver.SessionConfig{AccessMode: neo4jdriver.AccessModeWrite})
	defer session.Close(ctx)
	if _, err := session.Run(ctx, `MATCH (u:CurriculumUnit {id: $id}) DETACH DELETE u`,
		map[string]any{"id": id}); err != nil {
		return fmt.Errorf("delete unit from neo4j: %w", err)
	}
	return nil
}

// ── Prerequisite graph (Neo4j) ──────────────────────────────────

func (r *Repository) AddPrerequisite(ctx context.Context, unitID, prerequisiteUnitID string) error {
	session := r.neo4j.NewSession(ctx, neo4jdriver.SessionConfig{AccessMode: neo4jdriver.AccessModeWrite})
	defer session.Close(ctx)
	_, err := session.Run(ctx, `
		MATCH (u:CurriculumUnit {id: $unitId}), (p:CurriculumUnit {id: $prereqId})
		MERGE (p)-[:PREREQUISITE_OF]->(u)`,
		map[string]any{"unitId": unitID, "prereqId": prerequisiteUnitID},
	)
	if err != nil {
		return fmt.Errorf("add prerequisite: %w", err)
	}
	return nil
}

// Prerequisites returns the units that must be completed before unitID.
func (r *Repository) Prerequisites(ctx context.Context, unitID string) ([]PrerequisiteNode, error) {
	session := r.neo4j.NewSession(ctx, neo4jdriver.SessionConfig{AccessMode: neo4jdriver.AccessModeRead})
	defer session.Close(ctx)

	result, err := session.Run(ctx, `
		MATCH (p:CurriculumUnit)-[:PREREQUISITE_OF]->(u:CurriculumUnit {id: $unitId})
		RETURN p.id AS id, p.title AS title`,
		map[string]any{"unitId": unitID},
	)
	if err != nil {
		return nil, fmt.Errorf("query prerequisites: %w", err)
	}

	var nodes []PrerequisiteNode
	for result.Next(ctx) {
		rec := result.Record()
		id, _ := rec.Get("id")
		title, _ := rec.Get("title")
		nodes = append(nodes, PrerequisiteNode{ID: fmt.Sprint(id), Title: fmt.Sprint(title)})
	}
	if err := result.Err(); err != nil {
		return nil, fmt.Errorf("read prerequisites: %w", err)
	}
	return nodes, nil
}

type PrerequisiteNode struct {
	ID    string
	Title string
}

func scanUnit(row pgx.Row) (Unit, error) {
	var u Unit
	if err := row.Scan(&u.ID, &u.SubjectID, &u.Title, &u.Description, &u.OrderIndex, &u.CreatedAt); err != nil {
		return Unit{}, fmt.Errorf("scan unit: %w", err)
	}
	return u, nil
}
