package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	neo4jdriver "github.com/neo4j/neo4j-go-driver/v5/neo4j"

	"github.com/edugraph-ai/edugraph/internal/curriculum/dto"
)

var (
	ErrTopicNotFound        = errors.New("topic not found")
	ErrPrerequisiteCycle    = errors.New("prerequisite would create a cycle")
	ErrPrerequisiteNotFound = errors.New("prerequisite link not found")
)

// UnsyncedPrerequisite is one topic_prerequisites row whose Neo4j mirror
// isn't known-current (neo4j_written = false) -- either a best-effort sync
// failed at write time, or the row predates the neo4j_written column
// (V026) entirely. See Repository.ListUnsyncedPrerequisites and
// Service.ResyncPrerequisitesToNeo4j (feature 1.5).
type UnsyncedPrerequisite struct {
	TopicID     uuid.UUID
	PrereqID    uuid.UUID
	Weight      float64
	IsValidated bool
	InferMethod string
}

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
//
// inferMethod controls whether the link starts out validated: any
// human-authored provenance ("manual", "explicit", "moe_document") is
// confirmed immediately (reviewed_by/confirmed_at set to the caller/now);
// "ai_inferred" is left unconfirmed (both null) so it shows as
// IsValidated=false until a separate ValidatePrerequisite call reviews it
// -- the "validated vs. inferred" distinction (feature 1.4).
func (r *Repository) AddTopicPrerequisite(
	ctx context.Context, topicID, prereqID uuid.UUID, weight float64, inferMethod string, userID uuid.UUID,
) (*dto.PrerequisiteLink, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin add prerequisite tx: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

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

	if inferMethod == "" {
		inferMethod = "manual"
	}
	validated := inferMethod != "ai_inferred"
	var reviewedByParam *uuid.UUID
	var confirmedAtParam *time.Time
	if validated {
		reviewedByParam = &userID
		now := time.Now()
		confirmedAtParam = &now
	}

	isCrossGrade := topic.gradeLevel != prereq.gradeLevel
	_, err = tx.Exec(ctx, `
		INSERT INTO curriculum.topic_prerequisites
			(topic_id, prerequisite_id, weight, is_cross_grade, infer_method, reviewed_by, confirmed_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		ON CONFLICT (topic_id, prerequisite_id) DO UPDATE SET
			weight = EXCLUDED.weight,
			is_cross_grade = EXCLUDED.is_cross_grade,
			infer_method = EXCLUDED.infer_method,
			reviewed_by = EXCLUDED.reviewed_by,
			confirmed_at = EXCLUDED.confirmed_at,
			neo4j_written = false
	`, topicID, prereqID, weight, isCrossGrade, inferMethod, reviewedByParam, confirmedAtParam)
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
		InferMethod:         inferMethod,
		IsValidated:         validated,
	}, nil
}

// ValidatePrerequisite confirms an existing prerequisite link -- the
// counterpart to an "ai_inferred" link created unconfirmed by
// AddTopicPrerequisite. Sets reviewed_by/confirmed_at (so IsValidated
// becomes true) and clears neo4j_written so the caller knows to re-sync
// the Neo4j relationship's isValidated property.
func (r *Repository) ValidatePrerequisite(ctx context.Context, topicID, prereqID, userID uuid.UUID) (*dto.PrerequisiteLink, error) {
	var l dto.PrerequisiteLink
	err := r.pool.QueryRow(ctx, `
		UPDATE curriculum.topic_prerequisites tp
		SET reviewed_by = $3, confirmed_at = now(), neo4j_written = false
		FROM curriculum.topics t, curriculum.topics p
		WHERE tp.topic_id = $1 AND tp.prerequisite_id = $2
		  AND t.id = tp.topic_id AND p.id = tp.prerequisite_id
		RETURNING t.id, t.title_en, p.id, p.title_en, p.grade_level, tp.weight, tp.is_cross_grade, tp.infer_method
	`, topicID, prereqID, userID).Scan(
		&l.TopicID, &l.TopicTitle, &l.PrerequisiteTopicID, &l.PrerequisiteTitle,
		&l.PrerequisiteGrade, &l.Weight, &l.IsCrossGrade, &l.InferMethod,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrPrerequisiteNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("validate prerequisite: %w", err)
	}
	l.IsValidated = true
	return &l, nil
}

// SyncPrerequisiteToNeo4j mirrors one prerequisite edge as
// (:Topic {id})-[:HAS_PREREQUISITE]->(:Topic {id}), the edge shape the
// gap-analysis root-cause walk and the study-plan topological sort read.
// MERGE on both endpoints: a topic whose curriculum was approved before
// graph sync existed still gets a (bare) node, and re-running is safe --
// same convention as syncCurriculumGraph. Best-effort by contract: the
// caller reports failure in the response instead of rolling back Postgres,
// and is responsible for calling MarkPrerequisiteSynced afterwards.
//
// isValidated/inferMethod are mirrored as relationship properties so any
// graph-side consumer (or a future Cypher query) can filter by confidence
// without a round-trip back to Postgres.
func (r *Repository) SyncPrerequisiteToNeo4j(ctx context.Context, topicID, prereqID uuid.UUID, weight float64, isValidated bool, inferMethod string) error {
	session := r.neo4j.NewSession(ctx, neo4jdriver.SessionConfig{AccessMode: neo4jdriver.AccessModeWrite})
	defer session.Close(ctx)

	_, err := session.Run(ctx, `
		MERGE (t:Topic {id: $topicId})
		MERGE (p:Topic {id: $prereqId})
		MERGE (t)-[rel:HAS_PREREQUISITE]->(p)
		SET rel.weight = $weight, rel.isValidated = $isValidated, rel.inferMethod = $inferMethod
	`, map[string]any{
		"topicId":     topicID.String(),
		"prereqId":    prereqID.String(),
		"weight":      weight,
		"isValidated": isValidated,
		"inferMethod": inferMethod,
	})
	if err != nil {
		return fmt.Errorf("sync prerequisite to neo4j: %w", err)
	}
	return nil
}

// MarkPrerequisiteSynced flips neo4j_written = true after a successful
// SyncPrerequisiteToNeo4j call (feature 1.5's sync-status tracking).
func (r *Repository) MarkPrerequisiteSynced(ctx context.Context, topicID, prereqID uuid.UUID) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE curriculum.topic_prerequisites SET neo4j_written = true WHERE topic_id = $1 AND prerequisite_id = $2`,
		topicID, prereqID,
	)
	if err != nil {
		return fmt.Errorf("mark prerequisite synced: %w", err)
	}
	return nil
}

// ListUnsyncedPrerequisites returns every link with neo4j_written = false
// -- both ones whose best-effort sync failed at write time and ones that
// predate the neo4j_written column (V026) entirely. Backs the bulk resync
// endpoint (feature 1.5): the only way today to bring Neo4j back in step
// with Postgres if it was ever wiped, rebuilt, or just fell behind.
func (r *Repository) ListUnsyncedPrerequisites(ctx context.Context) ([]UnsyncedPrerequisite, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT topic_id, prerequisite_id, weight, (confirmed_at IS NOT NULL), infer_method
		FROM curriculum.topic_prerequisites
		WHERE NOT neo4j_written
	`)
	if err != nil {
		return nil, fmt.Errorf("list unsynced prerequisites: %w", err)
	}
	defer rows.Close()

	out := make([]UnsyncedPrerequisite, 0)
	for rows.Next() {
		var u UnsyncedPrerequisite
		if err := rows.Scan(&u.TopicID, &u.PrereqID, &u.Weight, &u.IsValidated, &u.InferMethod); err != nil {
			return nil, fmt.Errorf("scan unsynced prerequisite: %w", err)
		}
		out = append(out, u)
	}
	return out, rows.Err()
}

// ListTopicPrerequisites returns a topic's direct prerequisites.
func (r *Repository) ListTopicPrerequisites(ctx context.Context, topicID uuid.UUID) ([]dto.PrerequisiteLink, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT t.id, t.title_en, p.id, p.title_en, p.grade_level,
		       tp.weight, tp.is_cross_grade, tp.infer_method, (tp.confirmed_at IS NOT NULL)
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
			&l.PrerequisiteGrade, &l.Weight, &l.IsCrossGrade, &l.InferMethod, &l.IsValidated,
		); err != nil {
			return nil, fmt.Errorf("scan prerequisite link: %w", err)
		}
		out = append(out, l)
	}
	return out, rows.Err()
}
