package repository

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	neo4jdriver "github.com/neo4j/neo4j-go-driver/v5/neo4j"
)

var ErrNotFound = errors.New("not found")

type CareerPath struct {
	ID               string
	Title            string
	Description      *string
	RequiredSubjects []string
	CreatedAt        time.Time
}

type Match struct {
	CareerPathID string
	Title        string
	Score        float64
}

type Repository struct {
	pool  *pgxpool.Pool
	neo4j neo4jdriver.DriverWithContext
}

func New(pool *pgxpool.Pool, neo4j neo4jdriver.DriverWithContext) *Repository {
	return &Repository{pool: pool, neo4j: neo4j}
}

func (r *Repository) Create(ctx context.Context, title string, description *string, requiredSubjects []string) (CareerPath, error) {
	subjectsJSON, err := json.Marshal(requiredSubjects)
	if err != nil {
		return CareerPath{}, fmt.Errorf("marshal required subjects: %w", err)
	}

	const q = `INSERT INTO career_paths (title, description, required_subjects) VALUES ($1, $2, $3)
		RETURNING id, title, description, required_subjects, created_at`
	cp, err := scanCareerPath(r.pool.QueryRow(ctx, q, title, description, subjectsJSON))
	if err != nil {
		return CareerPath{}, err
	}

	session := r.neo4j.NewSession(ctx, neo4jdriver.SessionConfig{AccessMode: neo4jdriver.AccessModeWrite})
	defer session.Close(ctx)
	if _, err := session.Run(ctx, `MERGE (c:CareerPath {id: $id}) SET c.title = $title`,
		map[string]any{"id": cp.ID, "title": cp.Title}); err != nil {
		return CareerPath{}, fmt.Errorf("mirror career path to neo4j: %w", err)
	}

	return cp, nil
}

func (r *Repository) List(ctx context.Context) ([]CareerPath, error) {
	const q = `SELECT id, title, description, required_subjects, created_at FROM career_paths ORDER BY title`
	rows, err := r.pool.Query(ctx, q)
	if err != nil {
		return nil, fmt.Errorf("list career paths: %w", err)
	}
	defer rows.Close()

	var paths []CareerPath
	for rows.Next() {
		cp, err := scanCareerPath(rows)
		if err != nil {
			return nil, err
		}
		paths = append(paths, cp)
	}
	return paths, nil
}

func (r *Repository) GetByID(ctx context.Context, id string) (CareerPath, error) {
	const q = `SELECT id, title, description, required_subjects, created_at FROM career_paths WHERE id = $1`
	cp, err := scanCareerPath(r.pool.QueryRow(ctx, q, id))
	if errors.Is(err, pgx.ErrNoRows) {
		return CareerPath{}, ErrNotFound
	}
	return cp, err
}

// StudentSubjectAverages returns the student's average assessment score per
// subject, used as the input signal for career matching.
func (r *Repository) StudentSubjectAverages(ctx context.Context, studentID string) (map[string]float64, error) {
	const q = `
		SELECT sub.name, avg(ar.score)
		FROM assessment_results ar
		JOIN assessments a ON a.id = ar.assessment_id
		JOIN subjects sub ON sub.id = a.subject_id
		WHERE ar.student_id = $1
		GROUP BY sub.name`

	rows, err := r.pool.Query(ctx, q, studentID)
	if err != nil {
		return nil, fmt.Errorf("query subject averages: %w", err)
	}
	defer rows.Close()

	averages := make(map[string]float64)
	for rows.Next() {
		var subject string
		var avg float64
		if err := rows.Scan(&subject, &avg); err != nil {
			return nil, fmt.Errorf("scan subject average: %w", err)
		}
		averages[subject] = avg
	}
	return averages, nil
}

// SaveMatches persists generated matches and mirrors them as Neo4j edges so
// the graph can be traversed for subject -> career recommendations.
func (r *Repository) SaveMatches(ctx context.Context, studentID string, matches []Match) error {
	for _, m := range matches {
		const q = `INSERT INTO career_matches (student_id, career_path_id, match_score)
			VALUES ($1, $2, $3)
			ON CONFLICT (student_id, career_path_id)
			DO UPDATE SET match_score = $3, generated_at = now()`
		if _, err := r.pool.Exec(ctx, q, studentID, m.CareerPathID, m.Score); err != nil {
			return fmt.Errorf("save match: %w", err)
		}
	}

	session := r.neo4j.NewSession(ctx, neo4jdriver.SessionConfig{AccessMode: neo4jdriver.AccessModeWrite})
	defer session.Close(ctx)
	for _, m := range matches {
		_, err := session.Run(ctx, `
			MATCH (s:Student {id: $studentId}), (c:CareerPath {id: $careerId})
			MERGE (s)-[r:MATCHED_TO]->(c)
			SET r.score = $score`,
			map[string]any{"studentId": studentID, "careerId": m.CareerPathID, "score": m.Score},
		)
		if err != nil {
			return fmt.Errorf("mirror match to neo4j: %w", err)
		}
	}
	return nil
}

func (r *Repository) ListMatches(ctx context.Context, studentID string) ([]Match, error) {
	const q = `
		SELECT cm.career_path_id, cp.title, cm.match_score
		FROM career_matches cm
		JOIN career_paths cp ON cp.id = cm.career_path_id
		WHERE cm.student_id = $1
		ORDER BY cm.match_score DESC`

	rows, err := r.pool.Query(ctx, q, studentID)
	if err != nil {
		return nil, fmt.Errorf("list matches: %w", err)
	}
	defer rows.Close()

	var matches []Match
	for rows.Next() {
		var m Match
		if err := rows.Scan(&m.CareerPathID, &m.Title, &m.Score); err != nil {
			return nil, fmt.Errorf("scan match: %w", err)
		}
		matches = append(matches, m)
	}
	return matches, nil
}

func scanCareerPath(row pgx.Row) (CareerPath, error) {
	var cp CareerPath
	var rawSubjects []byte
	if err := row.Scan(&cp.ID, &cp.Title, &cp.Description, &rawSubjects, &cp.CreatedAt); err != nil {
		return CareerPath{}, fmt.Errorf("scan career path: %w", err)
	}
	_ = json.Unmarshal(rawSubjects, &cp.RequiredSubjects)
	return cp, nil
}
