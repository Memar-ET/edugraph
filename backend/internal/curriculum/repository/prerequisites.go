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
	EdgeType    string
	Confidence  *float64
}

// hardDependencyEdgeTypes are the only edge types that participate in
// cycle detection and every prerequisite-chain traversal (topological
// sort, gap-analysis root-cause walk, class-heatmap cross-grade alert).
// The other five EG-GCKT edge types (recommended_before/related_to/
// similar_to/supports/alternative_to) are soft associations -- e.g.
// similar_to is legitimately symmetric (A similar_to B and B similar_to A
// is correct, not a cycle bug) and must never hard-block an insert or be
// walked as if it were a dependency chain.
func isHardDependencyEdge(edgeType string) bool {
	return edgeType == "requires" || edgeType == "strongly_requires"
}

// AddPrerequisiteParams is Repository.AddTopicPrerequisite's input --
// grouped into a struct rather than a growing positional parameter list
// since EG-GCKT added Confidence/Evidence/EdgeType/CreatedByModel/
// CurriculumVersion on top of the original Weight/InferMethod pair.
type AddPrerequisiteParams struct {
	TopicID           uuid.UUID
	PrereqID          uuid.UUID
	Weight            float64
	InferMethod       string
	EdgeType          string
	Confidence        *float64
	Evidence          *string
	CreatedByModel    *string
	CurriculumVersion *string
	UserID            uuid.UUID
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

// AddTopicPrerequisite records a typed prerequisite edge (spec section
// 6.2 -- requires/strongly_requires/recommended_before/related_to/
// similar_to/supports/alternative_to) in curriculum.topic_prerequisites.
// Upsert on (topic_id, prerequisite_id, edge_type) so re-adding an
// existing link just updates its weight/confidence -- idempotent, like
// every other write in this repository -- while still allowing e.g.
// "A requires B" and "A similar_to B" to coexist as distinct rows.
//
// The cycle check runs in the same transaction as the insert, but only
// for hard-dependency edge types (requires/strongly_requires) -- adding
// T -[requires]-> P is rejected if T is already reachable FROM P through
// existing hard-dependency edges (P transitively requires T), which would
// deadlock the topological sort the study-plan generator runs. Soft edge
// types (similar_to, related_to, ...) bypass this check entirely: a
// symmetric similar_to pair is correct, not a cycle bug.
//
// inferMethod controls whether the link starts out validated: any
// human-authored provenance ("manual", "explicit", "moe_document") is
// confirmed immediately (reviewed_by/confirmed_at set to the caller/now);
// "ai_inferred" is left unconfirmed (both null) so it shows as
// IsValidated=false until a separate ValidatePrerequisite call reviews it
// -- the "validated vs. inferred" distinction (feature 1.4).
func (r *Repository) AddTopicPrerequisite(ctx context.Context, p AddPrerequisiteParams) (*dto.PrerequisiteLink, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin add prerequisite tx: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	topic, err := r.fetchTopicCore(ctx, tx, p.TopicID)
	if err != nil {
		return nil, err
	}
	prereq, err := r.fetchTopicCore(ctx, tx, p.PrereqID)
	if err != nil {
		return nil, err
	}

	edgeType := p.EdgeType
	if edgeType == "" {
		edgeType = "requires"
	}

	if isHardDependencyEdge(edgeType) {
		var cycle bool
		err = tx.QueryRow(ctx, `
			WITH RECURSIVE chain AS (
				SELECT tp.prerequisite_id
				FROM curriculum.topic_prerequisites tp
				WHERE tp.topic_id = $1 AND tp.edge_type IN ('requires', 'strongly_requires')
				UNION
				SELECT tp.prerequisite_id
				FROM curriculum.topic_prerequisites tp
				JOIN chain c ON tp.topic_id = c.prerequisite_id
				WHERE tp.edge_type IN ('requires', 'strongly_requires')
			)
			SELECT EXISTS (SELECT 1 FROM chain WHERE prerequisite_id = $2)
		`, p.PrereqID, p.TopicID).Scan(&cycle)
		if err != nil {
			return nil, fmt.Errorf("prerequisite cycle check: %w", err)
		}
		if cycle {
			return nil, ErrPrerequisiteCycle
		}
	}

	inferMethod := p.InferMethod
	if inferMethod == "" {
		inferMethod = "manual"
	}
	validated := inferMethod != "ai_inferred"
	var reviewedByParam *uuid.UUID
	var confirmedAtParam *time.Time
	if validated {
		reviewedByParam = &p.UserID
		now := time.Now()
		confirmedAtParam = &now
	}

	isCrossGrade := topic.gradeLevel != prereq.gradeLevel
	var edgeID uuid.UUID
	err = tx.QueryRow(ctx, `
		INSERT INTO curriculum.topic_prerequisites
			(topic_id, prerequisite_id, weight, is_cross_grade, infer_method, reviewed_by, confirmed_at,
			 edge_type, confidence, evidence, created_by, created_by_model, curriculum_version)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
		ON CONFLICT (topic_id, prerequisite_id, edge_type) DO UPDATE SET
			weight = EXCLUDED.weight,
			is_cross_grade = EXCLUDED.is_cross_grade,
			infer_method = EXCLUDED.infer_method,
			reviewed_by = EXCLUDED.reviewed_by,
			confirmed_at = EXCLUDED.confirmed_at,
			confidence = EXCLUDED.confidence,
			evidence = EXCLUDED.evidence,
			curriculum_version = EXCLUDED.curriculum_version,
			neo4j_written = false
		RETURNING id
	`, p.TopicID, p.PrereqID, p.Weight, isCrossGrade, inferMethod, reviewedByParam, confirmedAtParam,
		edgeType, p.Confidence, p.Evidence, p.UserID, p.CreatedByModel, p.CurriculumVersion,
	).Scan(&edgeID)
	if err != nil {
		return nil, fmt.Errorf("insert topic prerequisite: %w", err)
	}

	if _, err := tx.Exec(ctx, `
		INSERT INTO curriculum.prerequisite_review_history (prerequisite_edge_id, action, new_values, reviewed_by)
		VALUES ($1, 'created', jsonb_build_object('edgeType', $2::text, 'weight', $3::float8, 'confidence', $4::float8, 'inferMethod', $5::text), $6)
	`, edgeID, edgeType, p.Weight, p.Confidence, inferMethod, p.UserID); err != nil {
		return nil, fmt.Errorf("record prerequisite review history: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit add prerequisite: %w", err)
	}

	return &dto.PrerequisiteLink{
		TopicID:             p.TopicID.String(),
		TopicTitle:          topic.titleEn,
		PrerequisiteTopicID: p.PrereqID.String(),
		PrerequisiteTitle:   prereq.titleEn,
		PrerequisiteGrade:   prereq.gradeLevel,
		Weight:              p.Weight,
		IsCrossGrade:        isCrossGrade,
		InferMethod:         inferMethod,
		IsValidated:         validated,
		EdgeType:            edgeType,
		Confidence:          p.Confidence,
		Evidence:            p.Evidence,
		CurriculumVersion:   p.CurriculumVersion,
	}, nil
}

// ValidatePrerequisite confirms an existing prerequisite link -- the
// counterpart to an "ai_inferred" link created unconfirmed by
// AddTopicPrerequisite. Sets reviewed_by/confirmed_at (so IsValidated
// becomes true) and clears neo4j_written so the caller knows to re-sync
// the Neo4j relationship's isValidated property. edgeType disambiguates
// which of possibly several (topic, prerequisite) edges (one per type) is
// being validated -- defaults to "requires" for callers that don't care
// (the pre-EG-GCKT single-edge-type behavior).
func (r *Repository) ValidatePrerequisite(ctx context.Context, topicID, prereqID, userID uuid.UUID, edgeType string) (*dto.PrerequisiteLink, error) {
	if edgeType == "" {
		edgeType = "requires"
	}
	var l dto.PrerequisiteLink
	var edgeID uuid.UUID
	err := r.pool.QueryRow(ctx, `
		UPDATE curriculum.topic_prerequisites tp
		SET reviewed_by = $4, confirmed_at = now(), neo4j_written = false
		FROM curriculum.topics t, curriculum.topics p
		WHERE tp.topic_id = $1 AND tp.prerequisite_id = $2 AND tp.edge_type = $3
		  AND t.id = tp.topic_id AND p.id = tp.prerequisite_id
		RETURNING tp.id, t.id, t.title_en, p.id, p.title_en, p.grade_level, tp.weight, tp.is_cross_grade,
		          tp.infer_method, tp.edge_type, tp.confidence, tp.evidence
	`, topicID, prereqID, edgeType, userID).Scan(
		&edgeID, &l.TopicID, &l.TopicTitle, &l.PrerequisiteTopicID, &l.PrerequisiteTitle,
		&l.PrerequisiteGrade, &l.Weight, &l.IsCrossGrade, &l.InferMethod,
		&l.EdgeType, &l.Confidence, &l.Evidence,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrPrerequisiteNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("validate prerequisite: %w", err)
	}
	l.IsValidated = true

	if _, err := r.pool.Exec(ctx, `
		INSERT INTO curriculum.prerequisite_review_history (prerequisite_edge_id, action, reviewed_by)
		VALUES ($1, 'validated', $2)
	`, edgeID, userID); err != nil {
		return nil, fmt.Errorf("record validation review history: %w", err)
	}

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
// isValidated/inferMethod/edgeType/confidence are mirrored as relationship
// properties so any graph-side consumer (or a future Cypher query) can
// filter by confidence or edge type without a round-trip back to
// Postgres. Kept as a single :HAS_PREREQUISITE relationship type with a
// distinguishing edgeType property, rather than seven Neo4j relationship
// types, so every existing traversal (GetSubjectGraph, the class-heatmap
// root-cause walk) keeps working against one relationship type and only
// needs a property filter added where it must restrict to hard
// dependencies -- see CohortRootCauseNeo4j in the assessment package.
func (r *Repository) SyncPrerequisiteToNeo4j(ctx context.Context, topicID, prereqID uuid.UUID, weight float64, isValidated bool, inferMethod, edgeType string, confidence *float64) error {
	session := r.neo4j.NewSession(ctx, neo4jdriver.SessionConfig{AccessMode: neo4jdriver.AccessModeWrite})
	defer session.Close(ctx)

	if edgeType == "" {
		edgeType = "requires"
	}
	params := map[string]any{
		"topicId":     topicID.String(),
		"prereqId":    prereqID.String(),
		"weight":      weight,
		"isValidated": isValidated,
		"inferMethod": inferMethod,
		"edgeType":    edgeType,
		"confidence":  nil,
	}
	if confidence != nil {
		params["confidence"] = *confidence
	}

	_, err := session.Run(ctx, `
		MERGE (t:Topic {id: $topicId})
		MERGE (p:Topic {id: $prereqId})
		MERGE (t)-[rel:HAS_PREREQUISITE {edgeType: $edgeType}]->(p)
		SET rel.weight = $weight, rel.isValidated = $isValidated, rel.inferMethod = $inferMethod,
		    rel.confidence = $confidence
	`, params)
	if err != nil {
		return fmt.Errorf("sync prerequisite to neo4j: %w", err)
	}
	return nil
}

// MarkPrerequisiteSynced flips neo4j_written = true after a successful
// SyncPrerequisiteToNeo4j call (feature 1.5's sync-status tracking).
func (r *Repository) MarkPrerequisiteSynced(ctx context.Context, topicID, prereqID uuid.UUID, edgeType string) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE curriculum.topic_prerequisites SET neo4j_written = true WHERE topic_id = $1 AND prerequisite_id = $2 AND edge_type = $3`,
		topicID, prereqID, edgeType,
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
		SELECT topic_id, prerequisite_id, weight, (confirmed_at IS NOT NULL), infer_method, edge_type, confidence
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
		if err := rows.Scan(&u.TopicID, &u.PrereqID, &u.Weight, &u.IsValidated, &u.InferMethod, &u.EdgeType, &u.Confidence); err != nil {
			return nil, fmt.Errorf("scan unsynced prerequisite: %w", err)
		}
		out = append(out, u)
	}
	return out, rows.Err()
}

// ListTopicPrerequisites returns a topic's direct prerequisites, of every
// edge type.
func (r *Repository) ListTopicPrerequisites(ctx context.Context, topicID uuid.UUID) ([]dto.PrerequisiteLink, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT t.id, t.title_en, p.id, p.title_en, p.grade_level,
		       tp.weight, tp.is_cross_grade, tp.infer_method, (tp.confirmed_at IS NOT NULL),
		       tp.edge_type, tp.confidence, tp.evidence, tp.curriculum_version
		FROM curriculum.topic_prerequisites tp
		JOIN curriculum.topics t ON t.id = tp.topic_id
		JOIN curriculum.topics p ON p.id = tp.prerequisite_id
		WHERE tp.topic_id = $1
		ORDER BY tp.edge_type, tp.weight DESC
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
			&l.EdgeType, &l.Confidence, &l.Evidence, &l.CurriculumVersion,
		); err != nil {
			return nil, fmt.Errorf("scan prerequisite link: %w", err)
		}
		out = append(out, l)
	}
	return out, rows.Err()
}

// ListPrerequisiteReviewHistory returns the full append-only history for
// one prerequisite edge, newest first -- backs
// GET /curriculum/topics/:id/prerequisites/:prereqId/history (Milestone 11).
func (r *Repository) ListPrerequisiteReviewHistory(ctx context.Context, topicID, prereqID uuid.UUID, edgeType string) ([]dto.PrerequisiteReviewHistoryEntry, error) {
	if edgeType == "" {
		edgeType = "requires"
	}
	rows, err := r.pool.Query(ctx, `
		SELECT h.id, h.action, h.previous_values::text, h.new_values::text, h.reviewed_by::text, h.reviewed_at, h.notes
		FROM curriculum.prerequisite_review_history h
		JOIN curriculum.topic_prerequisites tp ON tp.id = h.prerequisite_edge_id
		WHERE tp.topic_id = $1 AND tp.prerequisite_id = $2 AND tp.edge_type = $3
		ORDER BY h.reviewed_at DESC
	`, topicID, prereqID, edgeType)
	if err != nil {
		return nil, fmt.Errorf("list prerequisite review history: %w", err)
	}
	defer rows.Close()

	out := make([]dto.PrerequisiteReviewHistoryEntry, 0)
	for rows.Next() {
		var e dto.PrerequisiteReviewHistoryEntry
		var reviewedAt time.Time
		if err := rows.Scan(&e.ID, &e.Action, &e.PreviousValues, &e.NewValues, &e.ReviewedBy, &reviewedAt, &e.Notes); err != nil {
			return nil, fmt.Errorf("scan prerequisite review history: %w", err)
		}
		e.ReviewedAt = reviewedAt.Format(time.RFC3339)
		out = append(out, e)
	}
	return out, rows.Err()
}
