package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/edugraph-ai/edugraph/internal/curriculum/dto"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	neo4jdriver "github.com/neo4j/neo4j-go-driver/v5/neo4j"
)

var (
	ErrTopicNotFound     = errors.New("topic not found")
	ErrPrerequisiteCycle = errors.New("prerequisite would create a cycle")
)

type topicCore struct {
	id         uuid.UUID
	titleEn    string
	gradeLevel int
}

func (r *Repository) fetchTopicCore(ctx context.Context, tx pgx.Tx, id uuid.UUID) (*topicCore, error) {
	var t topicCore
	err := tx.QueryRow(ctx,
		`SELECT id, title_en, grade_level FROM curriculum.topics WHERE id = $1`, id,
	).Scan(&t.id, &t.titleEn, &t.gradeLevel)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrTopicNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("fetch topic %s: %w", id, err)
	}
	return &t, nil
}

// AddTopicPrerequisite records "topic requires prerequisite" in
// curriculum.topic_prerequisites. Upsert on (topic_id, prerequisite_id)
// so re-adding an existing link just updates its weight -- idempotent,
// like every other write in this repository.
//
// The cycle check runs in the same transaction as the insert: adding
// T -> P is rejected if T is already reachable FROM P through existing
// prerequisite edges (P transitively requires T), which would deadlock
// the topological sort the study-plan generator (Capability 3B) runs.
func (r *Repository) AddTopicPrerequisite(
	ctx context.Context, topicID, prereqID uuid.UUID, weight float64, reviewedBy uuid.UUID,
) (*dto.PrerequisiteLink, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin add prerequisite tx: %w", err)
	}
	defer tx.Rollback(ctx)

	topic, err := r.fetchTopicCore(ctx, tx, topicID)
	if err != nil {
		return nil, err
	}
	prereq, err := r.fetchTopicCore(ctx, tx, prereqID)
	if err != nil {
		return nil, err
	}

	var cycle bool
	err = tx.QueryRow(ctx, `
		WITH RECURSIVE chain AS (
			SELECT tp.prerequisite_id
			FROM curriculum.topic_prerequisites tp
			WHERE tp.topic_id = $1
			UNION
			SELECT tp.prerequisite_id
			FROM curriculum.topic_prerequisites tp
			JOIN chain c ON tp.topic_id = c.prerequisite_id
		)
		SELECT EXISTS (SELECT 1 FROM chain WHERE prerequisite_id = $2)
	`, prereqID, topicID).Scan(&cycle)
	if err != nil {
		return nil, fmt.Errorf("prerequisite cycle check: %w", err)
	}
	if cycle {
		return nil, ErrPrerequisiteCycle
	}

	isCrossGrade := topic.gradeLevel != prereq.gradeLevel
	_, err = tx.Exec(ctx, `
		INSERT INTO curriculum.topic_prerequisites
			(topic_id, prerequisite_id, weight, is_cross_grade, infer_method, reviewed_by, confirmed_at)
		VALUES ($1, $2, $3, $4, 'manual', $5, now())
		ON CONFLICT (topic_id, prerequisite_id) DO UPDATE SET
			weight = EXCLUDED.weight,
			is_cross_grade = EXCLUDED.is_cross_grade,
			reviewed_by = EXCLUDED.reviewed_by,
			confirmed_at = now()
	`, topicID, prereqID, weight, isCrossGrade, reviewedBy)
	if err != nil {
		return nil, fmt.Errorf("insert topic prerequisite: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit add prerequisite: %w", err)
	}

	return &dto.PrerequisiteLink{
		TopicID:             topicID.String(),
		TopicTitle:          topic.titleEn,
		PrerequisiteTopicID: prereqID.String(),
		PrerequisiteTitle:   prereq.titleEn,
		PrerequisiteGrade:   prereq.gradeLevel,
		Weight:              weight,
		IsCrossGrade:        isCrossGrade,
		InferMethod:         "manual",
	}, nil
}

// SyncPrerequisiteToNeo4j mirrors one prerequisite edge as
// (:Topic {id})-[:HAS_PREREQUISITE]->(:Topic {id}), the edge shape the
// gap-analysis root-cause walk and the study-plan topological sort read.
// MERGE on both endpoints: a topic whose curriculum was approved before
// graph sync existed still gets a (bare) node, and re-running is safe --
// same convention as syncCurriculumGraph. Best-effort by contract: the
// caller reports failure in the response instead of rolling back Postgres.
func (r *Repository) SyncPrerequisiteToNeo4j(ctx context.Context, topicID, prereqID uuid.UUID, weight float64) error {
	session := r.neo4j.NewSession(ctx, neo4jdriver.SessionConfig{AccessMode: neo4jdriver.AccessModeWrite})
	defer session.Close(ctx)

	_, err := session.Run(ctx, `
		MERGE (t:Topic {id: $topicId})
		MERGE (p:Topic {id: $prereqId})
		MERGE (t)-[rel:HAS_PREREQUISITE]->(p)
		SET rel.weight = $weight
	`, map[string]any{
		"topicId":  topicID.String(),
		"prereqId": prereqID.String(),
		"weight":   weight,
	})
	if err != nil {
		return fmt.Errorf("sync prerequisite to neo4j: %w", err)
	}
	return nil
}

// ListTopicPrerequisites returns a topic's direct prerequisites.
func (r *Repository) ListTopicPrerequisites(ctx context.Context, topicID uuid.UUID) ([]dto.PrerequisiteLink, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT t.id, t.title_en, p.id, p.title_en, p.grade_level,
		       tp.weight, tp.is_cross_grade, tp.infer_method
		FROM curriculum.topic_prerequisites tp
		JOIN curriculum.topics t ON t.id = tp.topic_id
		JOIN curriculum.topics p ON p.id = tp.prerequisite_id
		WHERE tp.topic_id = $1
		ORDER BY tp.weight DESC
	`, topicID)
	if err != nil {
		return nil, fmt.Errorf("list topic prerequisites: %w", err)
	}
	defer rows.Close()

	out := make([]dto.PrerequisiteLink, 0)
	for rows.Next() {
		var l dto.PrerequisiteLink
		if err := rows.Scan(
			&l.TopicID, &l.TopicTitle, &l.PrerequisiteTopicID, &l.PrerequisiteTitle,
			&l.PrerequisiteGrade, &l.Weight, &l.IsCrossGrade, &l.InferMethod,
		); err != nil {
			return nil, fmt.Errorf("scan prerequisite link: %w", err)
		}
		out = append(out, l)
	}
	return out, rows.Err()
}
