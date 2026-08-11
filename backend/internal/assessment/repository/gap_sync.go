package repository

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	neo4jdriver "github.com/neo4j/neo4j-go-driver/v5/neo4j"
)

type UnsyncedGapRecord struct {
	ID               uuid.UUID
	StudentID        uuid.UUID
	SchoolID         uuid.UUID
	TopicID          uuid.UUID
	Severity         float64
	RootCauseTopicID *uuid.UUID
	PrereqDepth      int
	StudentGrade     int
	SchoolName       string
}

// FetchUnsyncedGapRecords drives the Capability 4A mirror: gap_records'
// neo4j_written flag + partial index (V011) follow the same outbox shape
// as exam attempts/answers. The students/schools joins pull the props the
// School and Student nodes need (cohort grade, display name) so the
// heatmap query can scope a "class" as school + grade level.
func (r *Repository) FetchUnsyncedGapRecords(ctx context.Context, limit int) ([]UnsyncedGapRecord, error) {
	const q = `
		SELECT g.id, g.student_id, g.school_id, g.topic_id, g.severity_score,
		       g.root_cause_topic_id, COALESCE(g.prerequisite_depth, 0),
		       s.grade_level, COALESCE(sc.name, '')
		FROM students.gap_records g
		JOIN students s ON s.id = g.student_id
		LEFT JOIN schools sc ON sc.id = g.school_id
		WHERE NOT g.neo4j_written
		LIMIT $1
	`
	rows, err := r.pool.Query(ctx, q, limit)
	if err != nil {
		return nil, fmt.Errorf("fetch unsynced gap records: %w", err)
	}
	defer rows.Close()

	var out []UnsyncedGapRecord
	for rows.Next() {
		var g UnsyncedGapRecord
		if err := rows.Scan(&g.ID, &g.StudentID, &g.SchoolID, &g.TopicID, &g.Severity,
			&g.RootCauseTopicID, &g.PrereqDepth, &g.StudentGrade, &g.SchoolName); err != nil {
			return nil, fmt.Errorf("scan unsynced gap record: %w", err)
		}
		out = append(out, g)
	}
	return out, rows.Err()
}

// SyncGapToNeo4j MERGEs (:School)-[:ENROLLS]->(:Student)-[:STRUGGLED_WITH]->(:Topic).
// Several gap records can target the same (student, topic) pair (one per
// missed question, across exams), so the edge keeps the WORST severity
// seen rather than the last write. When 3A traced the miss to a
// prerequisite, the root-cause topic gets its own STRUGGLED_WITH edge
// flagged isRootCause -- that's what lets the class heatmap's cross-grade
// alert count cohort strugglers on a Grade N-1 topic no Grade N exam ever
// assessed directly.
func (r *Repository) SyncGapToNeo4j(ctx context.Context, g UnsyncedGapRecord) error {
	session := r.neo4j.NewSession(ctx, neo4jdriver.SessionConfig{AccessMode: neo4jdriver.AccessModeWrite})
	defer session.Close(ctx)

	_, err := session.Run(ctx, `
		MERGE (sch:School {id: $schoolId})
		SET sch.name = CASE WHEN $schoolName <> '' THEN $schoolName ELSE sch.name END
		MERGE (st:Student {id: $studentId})
		SET st.gradeLevel = $gradeLevel, st.schoolId = $schoolId
		MERGE (sch)-[:ENROLLS]->(st)
		MERGE (t:Topic {id: $topicId})
		MERGE (st)-[r:STRUGGLED_WITH]->(t)
		ON CREATE SET r.severity = $severity, r.isRootCause = false
		ON MATCH SET r.severity = CASE WHEN coalesce(r.severity, 0.0) < $severity THEN $severity ELSE r.severity END
		SET r.lastDetectedAt = datetime()
	`, map[string]any{
		"schoolId":   g.SchoolID.String(),
		"schoolName": g.SchoolName,
		"studentId":  g.StudentID.String(),
		"gradeLevel": int64(g.StudentGrade),
		"topicId":    g.TopicID.String(),
		"severity":   g.Severity,
	})
	if err != nil {
		return fmt.Errorf("sync gap %s to neo4j: %w", g.ID, err)
	}

	if g.RootCauseTopicID != nil {
		_, err := session.Run(ctx, `
			MATCH (st:Student {id: $studentId})
			MERGE (rc:Topic {id: $rootTopicId})
			MERGE (st)-[r:STRUGGLED_WITH]->(rc)
			ON CREATE SET r.severity = $severity
			ON MATCH SET r.severity = CASE WHEN coalesce(r.severity, 0.0) < $severity THEN $severity ELSE r.severity END
			SET r.isRootCause = true,
			    r.viaTopicId = $symptomTopicId,
			    r.prerequisiteDepth = $depth,
			    r.lastDetectedAt = datetime()
		`, map[string]any{
			"studentId":      g.StudentID.String(),
			"rootTopicId":    g.RootCauseTopicID.String(),
			"severity":       g.Severity,
			"symptomTopicId": g.TopicID.String(),
			"depth":          int64(g.PrereqDepth),
		})
		if err != nil {
			return fmt.Errorf("sync gap %s root cause to neo4j: %w", g.ID, err)
		}
	}
	return nil
}

func (r *Repository) MarkGapSynced(ctx context.Context, id uuid.UUID) error {
	_, err := r.pool.Exec(ctx, `UPDATE students.gap_records SET neo4j_written = true, updated_at = now() WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("mark gap record synced: %w", err)
	}
	return nil
}
