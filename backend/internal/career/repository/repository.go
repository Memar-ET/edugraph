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

// This repository targets careers.careers/careers.career_matches/
// careers.career_topic_requirements -- V011__updated_curriculum.sql
// renamed the old bare career_paths/career_matches tables to
// career_paths_v010/career_matches_v010 (archived, not dropped, per that
// migration's own comment) to make room for this newer schema. A prior
// version of this repository stayed pointed at the archived _v010
// names on the stated rationale that the newer schema was "completely
// unused." That rationale no longer holds: on the live Supabase
// database the _v010 tables were never actually created (confirmed via
// \dt -- career_paths_v010/career_matches_v010 don't exist there at
// all, only careers.careers/careers.career_matches/
// careers.career_topic_requirements do), so every query here failed
// outright with "relation does not exist" rather than the intended
// "legacy archived data" fallback. Migrated to the real schema below.
//
// careers.careers has no flat required_subjects column -- requirements
// are topic-level (careers.career_topic_requirements -> curriculum.
// topics), so RequiredSubjects here is derived as the distinct set of
// subject codes among a career's required topics. There is still no
// curation UI for career_topic_requirements, so today this is
// legitimately empty for every career (an honest empty list, not a
// fabricated one) until that curation path is built.

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

func (r *Repository) Create(ctx context.Context, title string, description *string, sector, minEduLevel string) (CareerPath, error) {
	const q = `INSERT INTO careers.careers (name_en, description, sector, min_edu_level)
		VALUES ($1, $2, $3, $4)
		RETURNING id, name_en, description, created_at`
	var cp CareerPath
	err := r.pool.QueryRow(ctx, q, title, description, sector, minEduLevel).Scan(&cp.ID, &cp.Title, &cp.Description, &cp.CreatedAt)
	if err != nil {
		return CareerPath{}, fmt.Errorf("create career: %w", err)
	}
	cp.RequiredSubjects = []string{}

	session := r.neo4j.NewSession(ctx, neo4jdriver.SessionConfig{AccessMode: neo4jdriver.AccessModeWrite})
	defer session.Close(ctx)
	if _, err := session.Run(ctx, `MERGE (c:CareerPath {id: $id}) SET c.title = $title`,
		map[string]any{"id": cp.ID, "title": cp.Title}); err != nil {
		return CareerPath{}, fmt.Errorf("mirror career path to neo4j: %w", err)
	}

	return cp, nil
}

// careerColumns backs both List and GetByID -- RequiredSubjects is
// derived as the distinct subject codes among a career's required topics
// (careers.career_topic_requirements -> curriculum.topics), since
// careers.careers has no flat required-subjects column. Legitimately
// empty today for every career: there is no curation UI yet for
// career_topic_requirements.
const careerColumns = `
	SELECT c.id, c.name_en, c.description, c.created_at,
	       COALESCE(array_agg(DISTINCT t.subject_code) FILTER (WHERE t.subject_code IS NOT NULL), '{}')
	FROM careers.careers c
	LEFT JOIN careers.career_topic_requirements r ON r.career_id = c.id
	LEFT JOIN curriculum.topics t ON t.id = r.topic_id
`

func (r *Repository) List(ctx context.Context) ([]CareerPath, error) {
	const q = careerColumns + " GROUP BY c.id, c.name_en, c.description, c.created_at ORDER BY c.name_en"
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
	const q = careerColumns + " WHERE c.id = $1 GROUP BY c.id, c.name_en, c.description, c.created_at"
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
		const q = `INSERT INTO careers.career_matches (student_id, career_path_id, match_score)
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
		SELECT cm.career_path_id, c.name_en, cm.match_score
		FROM careers.career_matches cm
		JOIN careers.careers c ON c.id = cm.career_path_id
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
	if err := row.Scan(&cp.ID, &cp.Title, &cp.Description, &cp.CreatedAt, &cp.RequiredSubjects); err != nil {
		return CareerPath{}, fmt.Errorf("scan career path: %w", err)
	}
	return cp, nil
}
