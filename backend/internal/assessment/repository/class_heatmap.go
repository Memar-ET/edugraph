package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	neo4jdriver "github.com/neo4j/neo4j-go-driver/v5/neo4j"
)

// Capability 4A: Class-Wide Gap Heatmap. Graph-first (the spec's Cypher
// over School-ENROLLS->Student-STRUGGLED_WITH->Topic, mirrored by
// syncworker via gap_sync.go), with students.gap_records as the
// system-of-record fallback -- same Neo4j->PG contract as 3A/3B.

const heatmapTopicLimit = 15

type HeatmapRow struct {
	TopicID            string
	Title              string
	StudentsStruggling int
	AvgSeverity        float64
}

type RootCauseRow struct {
	TopicID            string
	Title              string
	GradeLevel         int
	StudentsStruggling int
}

// ClassSize is the heatmap's denominator: enrolled students in this
// school+grade per Postgres (the system of record), NOT distinct students
// seen in the graph -- students with no exam data yet still count as part
// of the class.
func (r *Repository) ClassSize(ctx context.Context, schoolID uuid.UUID, gradeLevel int) (int, error) {
	var n int
	err := r.pool.QueryRow(ctx,
		`SELECT count(*) FROM students WHERE school_id = $1 AND grade_level = $2`,
		schoolID, gradeLevel,
	).Scan(&n)
	if err != nil {
		return 0, fmt.Errorf("count class size: %w", err)
	}
	return n, nil
}

// HeatmapNeo4j ranks topics by how many cohort students carry a
// STRUGGLED_WITH edge. Cohort = enrolled at the school AND same grade
// level; the Student.gradeLevel filter keeps a grade-12 student's
// root-cause edge on a grade-11 topic out of the grade-11 class's counts.
func (r *Repository) HeatmapNeo4j(ctx context.Context, schoolID uuid.UUID, subjectCode string, gradeLevel int) ([]HeatmapRow, error) {
	session := r.neo4j.NewSession(ctx, neo4jdriver.SessionConfig{AccessMode: neo4jdriver.AccessModeRead})
	defer session.Close(ctx)

	result, err := session.Run(ctx, `
		MATCH (:School {id: $schoolId})-[:ENROLLS]->(s:Student)-[r:STRUGGLED_WITH]->(t:Topic)
		WHERE t.subjectCode = $subjectCode
		  AND t.gradeLevel = $gradeLevel
		  AND s.gradeLevel = $gradeLevel
		RETURN t.id AS topicId, t.titleEn AS title,
		       count(DISTINCT s) AS studentsStruggling,
		       avg(r.severity) AS avgSeverity
		ORDER BY studentsStruggling DESC, avgSeverity DESC
		LIMIT $limit
	`, map[string]any{
		"schoolId":    schoolID.String(),
		"subjectCode": subjectCode,
		"gradeLevel":  int64(gradeLevel),
		"limit":       int64(heatmapTopicLimit),
	})
	if err != nil {
		return nil, fmt.Errorf("neo4j heatmap query: %w", err)
	}

	var rows []HeatmapRow
	for result.Next(ctx) {
		rec := result.Record()
		row := HeatmapRow{}
		if v, ok := rec.Get("topicId"); ok && v != nil {
			row.TopicID, _ = v.(string)
		}
		if v, ok := rec.Get("title"); ok && v != nil {
			row.Title, _ = v.(string)
		}
		if v, ok := rec.Get("studentsStruggling"); ok && v != nil {
			if n, ok := v.(int64); ok {
				row.StudentsStruggling = int(n)
			}
		}
		if v, ok := rec.Get("avgSeverity"); ok && v != nil {
			if f, ok := v.(float64); ok {
				row.AvgSeverity = f
			}
		}
		rows = append(rows, row)
	}
	if err := result.Err(); err != nil {
		return nil, fmt.Errorf("neo4j heatmap result: %w", err)
	}
	return rows, nil
}

// CohortRootCauseNeo4j is the cross-grade alert traversal: from a hot
// topic, walk HAS_PREREQUISITE up to 3 hops to LOWER-grade topics and
// pick the one the most cohort students also carry a STRUGGLED_WITH edge
// for (3A's isRootCause edges are what usually put those there). Returns
// nil, nil when no lower-grade prerequisite has any cohort strugglers.
//
// EG-GCKT (Milestone 0) added typed prerequisite edges -- edgeType
// defaults to 'requires' on every pre-existing row, but non-'requires'
// edges (similar_to, related_to, ...) are soft associations, not
// dependency chains, and must never be walked here as if fixing an
// upstream "similar" topic would relieve a downstream one. The path
// filter restricts every hop to the two hard-dependency edge types.
func (r *Repository) CohortRootCauseNeo4j(ctx context.Context, schoolID uuid.UUID, topicID string, gradeLevel int) (*RootCauseRow, error) {
	session := r.neo4j.NewSession(ctx, neo4jdriver.SessionConfig{AccessMode: neo4jdriver.AccessModeRead})
	defer session.Close(ctx)

	result, err := session.Run(ctx, `
		MATCH path = (t:Topic {id: $topicId})-[:HAS_PREREQUISITE*1..3]->(p:Topic)
		WHERE p.gradeLevel < t.gradeLevel
		  AND ALL(rel IN relationships(path) WHERE coalesce(rel.edgeType, 'requires') IN ['requires', 'strongly_requires'])
		MATCH (:School {id: $schoolId})-[:ENROLLS]->(s:Student)-[:STRUGGLED_WITH]->(p)
		WHERE s.gradeLevel = $gradeLevel
		RETURN p.id AS topicId, p.titleEn AS title, p.gradeLevel AS grade,
		       count(DISTINCT s) AS studentsStruggling
		ORDER BY studentsStruggling DESC
		LIMIT 1
	`, map[string]any{
		"topicId":    topicID,
		"schoolId":   schoolID.String(),
		"gradeLevel": int64(gradeLevel),
	})
	if err != nil {
		return nil, fmt.Errorf("neo4j root cause query: %w", err)
	}

	if !result.Next(ctx) {
		if err := result.Err(); err != nil {
			return nil, fmt.Errorf("neo4j root cause result: %w", err)
		}
		return nil, nil
	}
	rec := result.Record()
	row := &RootCauseRow{}
	if v, ok := rec.Get("topicId"); ok && v != nil {
		row.TopicID, _ = v.(string)
	}
	if v, ok := rec.Get("title"); ok && v != nil {
		row.Title, _ = v.(string)
	}
	if v, ok := rec.Get("grade"); ok && v != nil {
		if n, ok := v.(int64); ok {
			row.GradeLevel = int(n)
		}
	}
	if v, ok := rec.Get("studentsStruggling"); ok && v != nil {
		if n, ok := v.(int64); ok {
			row.StudentsStruggling = int(n)
		}
	}
	return row, nil
}

// HeatmapPG is the fallback aggregation over students.gap_records --
// covers Neo4j being down AND records the syncworker hasn't mirrored yet.
// resolved_at IS NULL keeps it to open gaps (nothing resolves gaps today,
// so this matches the graph, where edges are never deleted either).
func (r *Repository) HeatmapPG(ctx context.Context, schoolID uuid.UUID, subjectCode string, gradeLevel int) ([]HeatmapRow, error) {
	const q = `
		SELECT g.topic_id, t.title_en,
		       count(DISTINCT g.student_id) AS students_struggling,
		       avg(g.severity_score)::float8 AS avg_severity
		FROM students.gap_records g
		JOIN curriculum.topics t ON t.id = g.topic_id
		JOIN students s ON s.id = g.student_id
		WHERE g.school_id = $1
		  AND t.subject_code = $2
		  AND t.grade_level = $3
		  AND s.grade_level = $3
		  AND g.resolved_at IS NULL
		GROUP BY g.topic_id, t.title_en
		ORDER BY students_struggling DESC, avg_severity DESC
		LIMIT $4
	`
	rows, err := r.pool.Query(ctx, q, schoolID, subjectCode, gradeLevel, heatmapTopicLimit)
	if err != nil {
		return nil, fmt.Errorf("pg heatmap query: %w", err)
	}
	defer rows.Close()

	var out []HeatmapRow
	for rows.Next() {
		var row HeatmapRow
		var topicID uuid.UUID
		if err := rows.Scan(&topicID, &row.Title, &row.StudentsStruggling, &row.AvgSeverity); err != nil {
			return nil, fmt.Errorf("scan pg heatmap row: %w", err)
		}
		row.TopicID = topicID.String()
		out = append(out, row)
	}
	return out, rows.Err()
}

// CohortRootCausePG aggregates the per-student root causes 3A already
// attributed (gap_records.root_cause_topic_id) for this school+topic,
// keeping only lower-grade causes -- the same cross-grade condition the
// Neo4j traversal enforces via p.gradeLevel < t.gradeLevel.
func (r *Repository) CohortRootCausePG(ctx context.Context, schoolID uuid.UUID, topicID string, gradeLevel int) (*RootCauseRow, error) {
	const q = `
		SELECT g.root_cause_topic_id, t.title_en, t.grade_level,
		       count(DISTINCT g.student_id) AS students_struggling
		FROM students.gap_records g
		JOIN curriculum.topics t ON t.id = g.root_cause_topic_id
		WHERE g.school_id = $1
		  AND g.topic_id = $2
		  AND g.root_cause_topic_id IS NOT NULL
		  AND t.grade_level < $3
		  AND g.resolved_at IS NULL
		GROUP BY g.root_cause_topic_id, t.title_en, t.grade_level
		ORDER BY students_struggling DESC
		LIMIT 1
	`
	var row RootCauseRow
	var rcID uuid.UUID
	err := r.pool.QueryRow(ctx, q, schoolID, topicID, gradeLevel).Scan(
		&rcID, &row.Title, &row.GradeLevel, &row.StudentsStruggling,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("pg root cause query: %w", err)
	}
	row.TopicID = rcID.String()
	return &row, nil
}

// SyncTeacherToNeo4j keeps the teacher connected to their school in the
// graph: (:Teacher)-[:TEACHES_AT]->(:School), MERGE-keyed on the
// Postgres teachers.id like every other mirrored node. Called best-effort
// from the heatmap read path; school_admins have no teachers row and are
// skipped (nil error).
func (r *Repository) SyncTeacherToNeo4j(ctx context.Context, userID, schoolID uuid.UUID) error {
	var teacherID uuid.UUID
	var specialty *string
	err := r.pool.QueryRow(ctx,
		`SELECT id, subject_specialty FROM teachers WHERE user_id = $1`, userID,
	).Scan(&teacherID, &specialty)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("lookup teacher for graph sync: %w", err)
	}

	session := r.neo4j.NewSession(ctx, neo4jdriver.SessionConfig{AccessMode: neo4jdriver.AccessModeWrite})
	defer session.Close(ctx)

	params := map[string]any{
		"teacherId": teacherID.String(),
		"userId":    userID.String(),
		"schoolId":  schoolID.String(),
		"specialty": "",
	}
	if specialty != nil {
		params["specialty"] = *specialty
	}
	_, err = session.Run(ctx, `
		MERGE (tch:Teacher {id: $teacherId})
		SET tch.userId = $userId,
		    tch.subjectSpecialty = CASE WHEN $specialty <> '' THEN $specialty ELSE tch.subjectSpecialty END
		MERGE (sch:School {id: $schoolId})
		MERGE (tch)-[:TEACHES_AT]->(sch)
	`, params)
	if err != nil {
		return fmt.Errorf("sync teacher %s to neo4j: %w", teacherID, err)
	}
	return nil
}
