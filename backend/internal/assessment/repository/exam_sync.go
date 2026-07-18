package repository

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	neo4jdriver "github.com/neo4j/neo4j-go-driver/v5/neo4j"
)

type UnsyncedAttempt struct {
	ID         uuid.UUID
	StudentID  uuid.UUID
	ExamID     uuid.UUID
	TotalScore float64
	Percentage float64
	Passed     bool
}

// FetchUnsyncedAttempts only returns finalized attempts (submitted_at IS
// NOT NULL) -- totals aren't meaningful to mirror into Neo4j until
// RecomputeAttemptTotals has actually finalized them. Uses the
// idx_exam_attempts-adjacent "WHERE NOT neo4j_written" shape the schema
// was built for (see V011) but nothing queried until this worker.
func (r *Repository) FetchUnsyncedAttempts(ctx context.Context, limit int) ([]UnsyncedAttempt, error) {
	const q = `
		SELECT id, student_id, exam_id, total_score, percentage, passed
		FROM assessment.exam_attempts
		WHERE NOT neo4j_written AND submitted_at IS NOT NULL
		LIMIT $1
	`
	rows, err := r.pool.Query(ctx, q, limit)
	if err != nil {
		return nil, fmt.Errorf("fetch unsynced attempts: %w", err)
	}
	defer rows.Close()

	var out []UnsyncedAttempt
	for rows.Next() {
		var a UnsyncedAttempt
		if err := rows.Scan(&a.ID, &a.StudentID, &a.ExamID, &a.TotalScore, &a.Percentage, &a.Passed); err != nil {
			return nil, fmt.Errorf("scan unsynced attempt: %w", err)
		}
		out = append(out, a)
	}
	return out, rows.Err()
}

type UnsyncedAnswer struct {
	ID           uuid.UUID
	StudentID    uuid.UUID
	QuestionID   uuid.UUID
	MarksAwarded float64
	Passed       bool
}

// FetchUnsyncedAnswers only returns graded answers (marks_awarded IS NOT
// NULL) -- a still-pending answer has nothing meaningful to put on an
// ANSWERED relationship yet; it'll be picked up once a teacher grades it
// and RecomputeAttemptTotals leaves it ungraded no longer.
func (r *Repository) FetchUnsyncedAnswers(ctx context.Context, limit int) ([]UnsyncedAnswer, error) {
	const q = `
		SELECT id, student_id, question_id, marks_awarded, COALESCE(passed, false)
		FROM assessment.student_answers
		WHERE NOT neo4j_written AND marks_awarded IS NOT NULL
		LIMIT $1
	`
	rows, err := r.pool.Query(ctx, q, limit)
	if err != nil {
		return nil, fmt.Errorf("fetch unsynced answers: %w", err)
	}
	defer rows.Close()

	var out []UnsyncedAnswer
	for rows.Next() {
		var a UnsyncedAnswer
		if err := rows.Scan(&a.ID, &a.StudentID, &a.QuestionID, &a.MarksAwarded, &a.Passed); err != nil {
			return nil, fmt.Errorf("scan unsynced answer: %w", err)
		}
		out = append(out, a)
	}
	return out, rows.Err()
}

// SyncAttemptToNeo4j MERGEs (:Student)-[:ATTEMPTED]->(:Exam), same
// MERGE-by-Postgres-id idempotent-retry pattern as curriculum's
// syncCurriculumGraph.
func (r *Repository) SyncAttemptToNeo4j(ctx context.Context, a UnsyncedAttempt) error {
	session := r.neo4j.NewSession(ctx, neo4jdriver.SessionConfig{AccessMode: neo4jdriver.AccessModeWrite})
	defer session.Close(ctx)

	_, err := session.Run(ctx, `
		MERGE (s:Student {id: $studentId})
		MERGE (e:Exam {id: $examId})
		MERGE (s)-[r:ATTEMPTED]->(e)
		SET r.totalScore = $totalScore, r.percentage = $percentage, r.passed = $passed
	`, map[string]any{
		"studentId":  a.StudentID.String(),
		"examId":     a.ExamID.String(),
		"totalScore": a.TotalScore,
		"percentage": a.Percentage,
		"passed":     a.Passed,
	})
	if err != nil {
		return fmt.Errorf("sync attempt %s to neo4j: %w", a.ID, err)
	}
	return nil
}

// SyncAnswerToNeo4j MERGEs (:Student)-[:ANSWERED]->(:Question).
func (r *Repository) SyncAnswerToNeo4j(ctx context.Context, a UnsyncedAnswer) error {
	session := r.neo4j.NewSession(ctx, neo4jdriver.SessionConfig{AccessMode: neo4jdriver.AccessModeWrite})
	defer session.Close(ctx)

	_, err := session.Run(ctx, `
		MERGE (s:Student {id: $studentId})
		MERGE (q:Question {id: $questionId})
		MERGE (s)-[r:ANSWERED]->(q)
		SET r.marksAwarded = $marksAwarded, r.passed = $passed
	`, map[string]any{
		"studentId":    a.StudentID.String(),
		"questionId":   a.QuestionID.String(),
		"marksAwarded": a.MarksAwarded,
		"passed":       a.Passed,
	})
	if err != nil {
		return fmt.Errorf("sync answer %s to neo4j: %w", a.ID, err)
	}
	return nil
}

func (r *Repository) MarkAttemptSynced(ctx context.Context, id uuid.UUID) error {
	_, err := r.pool.Exec(ctx, `UPDATE assessment.exam_attempts SET neo4j_written = true, synced_at = now() WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("mark attempt synced: %w", err)
	}
	return nil
}

func (r *Repository) MarkAnswerSynced(ctx context.Context, id uuid.UUID) error {
	_, err := r.pool.Exec(ctx, `UPDATE assessment.student_answers SET neo4j_written = true WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("mark answer synced: %w", err)
	}
	return nil
}
