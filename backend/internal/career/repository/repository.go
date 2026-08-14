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

// This repository targets career_paths_v010/career_matches_v010, not the
// bare career_paths/career_matches names visible elsewhere in this file's
// SQL history -- V011__updated_curriculum.sql renamed (archived, not
// dropped) the originals to make room for a newer careers.careers/
// careers.career_topic_requirements/careers.career_matches schema
// (topic-level requirements + importance weighting, closer to the PRD's
// Neo4j-native design). That newer schema is completely unused anywhere
// in the codebase today -- confirmed by grep, not assumed -- so this
// repository was left pointed at a table name (career_paths) that V011
// had already renamed out from under it, which is the deeper reason
// "Generate Career Matches" was broken: every query here failed outright
// with "relation does not exist", not just a downstream 502. Migrating
// to the newer topic-level schema (which has no create/curation UI for
// career_topic_requirements yet either) is real, larger future work --
// out of scope for fixing the actually-broken feature; see the
// career_matcher/service.py docstring on the ai-service side for the
// same call.
//
// A since-renumbered V032__cleanup_old_curriculum_tables.sql (merged
// from a different work stream, originally V012) tried to DROP these
// two tables outright, on the stated but unmet precondition that this
// repository had already migrated off them -- see that migration's own
// comment. Do not drop career_paths_v010/career_matches_v010 without
// actually migrating this file to the careers.* schema first.

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

	const q = `INSERT INTO career_paths_v010 (title, description, required_subjects) VALUES ($1, $2, $3)
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
	const q = `SELECT id, title, description, required_subjects, created_at FROM career_paths_v010 ORDER BY title`
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
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list career paths: %w", err)
	}
	return paths, nil
}

func (r *Repository) GetByID(ctx context.Context, id string) (CareerPath, error) {
	const q = `SELECT id, title, description, required_subjects, created_at FROM career_paths_v010 WHERE id = $1`
	cp, err := scanCareerPath(r.pool.QueryRow(ctx, q, id))
	if errors.Is(err, pgx.ErrNoRows) {
		return CareerPath{}, ErrNotFound
	}
	return cp, err
}

// StudentIDByUserID resolves the caller's own students.id from their
// users.id (JWT subject) -- see service.GenerateMatches/Matches, which
// use this instead of trusting a client-supplied studentID, closing the
// IDOR where any authenticated account could read or trigger generation
// for another student's career matches by passing their id in the URL.
func (r *Repository) StudentIDByUserID(ctx context.Context, userID string) (string, error) {
	var studentID string
	err := r.pool.QueryRow(ctx, `SELECT id FROM students WHERE user_id = $1`, userID).Scan(&studentID)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", ErrNotFound
	}
	if err != nil {
		return "", fmt.Errorf("lookup student by user id: %w", err)
	}
	return studentID, nil
}

// StudentSubjectAverages returns the student's average exam percentage per
// subject family, used as the input signal for career matching.
//
// This used to query assessment_results/assessments/subjects -- the
// legacy public-schema tables from before the exam pipeline moved to
// assessment.exam_attempts/assessment.exams. Nothing writes to those
// legacy tables anymore (confirmed by grep, not assumed), so this always
// returned an empty map for every real student, making career matching
// fail with "no graded assessments" before it ever reached the (also
// broken) ai-service call. Fixed to read the tables the exam-taking flow
// actually writes to.
//
// curriculum.subjects.code is grade-coupled (e.g. "BIO7", "MATH9"), but
// a career's required_subjects is a grade-independent family (a career
// path applies across grades, and CareerPathsPage's own placeholder --
// "PHY, MATH" -- confirms the short, grade-stripped convention), so the
// trailing digits are stripped here to group grades of the same subject
// together.
func (r *Repository) StudentSubjectAverages(ctx context.Context, studentID string) (map[string]float64, error) {
	const q = `
		SELECT TRIM(TRAILING '0123456789' FROM e.subject_code) AS subject_family,
		       avg(a.percentage)::float8
		FROM assessment.exam_attempts a
		JOIN assessment.exams e ON e.id = a.exam_id
		WHERE a.student_id = $1 AND a.percentage IS NOT NULL
		GROUP BY subject_family`

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
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("query subject averages: %w", err)
	}
	return averages, nil
}

// SaveMatches persists generated matches and mirrors them as Neo4j edges so
// the graph can be traversed for subject -> career recommendations.
func (r *Repository) SaveMatches(ctx context.Context, studentID string, matches []Match) error {
	for _, m := range matches {
		const q = `INSERT INTO career_matches_v010 (student_id, career_path_id, match_score)
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
		FROM career_matches_v010 cm
		JOIN career_paths_v010 cp ON cp.id = cm.career_path_id
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
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list matches: %w", err)
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
