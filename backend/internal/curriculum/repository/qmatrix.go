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
	ErrQuestionNotFound = errors.New("question not found")
	ErrMappingNotFound  = errors.New("item-skill mapping not found")
)

// UnsyncedItemSkillMapping mirrors UnsyncedPrerequisite for the Q-matrix
// resync path (Milestone 9's governance UI / any operational catch-up).
type UnsyncedItemSkillMapping struct {
	ID          uuid.UUID
	QuestionID  uuid.UUID
	TopicID     uuid.UUID
	Relevance   float64
	IsValidated bool
}

// AddItemSkillMappingParams is Repository.AddItemSkillMapping's input,
// grouped the same way AddPrerequisiteParams is.
type AddItemSkillMappingParams struct {
	QuestionID       uuid.UUID
	TopicID          uuid.UUID
	CloCode          *string
	Relevance        float64
	CognitiveLevel   *string
	GenerationMethod string
	UserID           uuid.UUID
}

// AddItemSkillMapping records a new versioned Q-matrix entry (spec
// section 6.3). Unlike AddTopicPrerequisite's upsert-in-place, a Q-matrix
// entry is append-only-versioned: any existing current row for
// (question_id, topic_id) is superseded (is_current=false,
// superseded_at=now()) and a brand-new row becomes the current one --
// mirrors curriculum.subjects' version/is_current/superseded_at pattern
// (V025) so a calibration pass (Milestone 8) can always add a new
// version without destroying the history a governance replay needs
// (spec section 19).
//
// generationMethod "manual"/"teacher_confirmed" auto-validates
// (confirmed_at set) the same way infer_method "manual"/"explicit"/
// "moe_document" does for prerequisites; "ai_draft"/"embedding_auto"
// start unconfirmed.
func (r *Repository) AddItemSkillMapping(ctx context.Context, p AddItemSkillMappingParams) (*dto.ItemSkillMapping, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin add item-skill mapping tx: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var topicTitle string
	err = tx.QueryRow(ctx, `SELECT title_en FROM curriculum.topics WHERE id = $1`, p.TopicID).Scan(&topicTitle)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrTopicNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("fetch topic for item-skill mapping: %w", err)
	}

	var questionExists bool
	if err := tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM assessment.questions WHERE id = $1)`, p.QuestionID).Scan(&questionExists); err != nil {
		return nil, fmt.Errorf("check question exists: %w", err)
	}
	if !questionExists {
		return nil, ErrQuestionNotFound
	}

	generationMethod := p.GenerationMethod
	if generationMethod == "" {
		generationMethod = "manual"
	}
	validated := generationMethod == "manual" || generationMethod == "teacher_confirmed"

	var prevVersion int
	var prevID *uuid.UUID
	err = tx.QueryRow(ctx, `
		SELECT id, version FROM assessment.item_skill_mappings
		WHERE question_id = $1 AND topic_id = $2 AND is_current
	`, p.QuestionID, p.TopicID).Scan(&prevID, &prevVersion)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return nil, fmt.Errorf("check existing item-skill mapping: %w", err)
	}
	newVersion := prevVersion + 1

	if prevID != nil {
		if _, err := tx.Exec(ctx, `
			UPDATE assessment.item_skill_mappings SET is_current = FALSE, superseded_at = now() WHERE id = $1
		`, *prevID); err != nil {
			return nil, fmt.Errorf("supersede prior item-skill mapping: %w", err)
		}
	}

	var reviewedByParam *uuid.UUID
	var confirmedAtParam *time.Time
	if validated {
		reviewedByParam = &p.UserID
		now := time.Now()
		confirmedAtParam = &now
	}

	var id uuid.UUID
	err = tx.QueryRow(ctx, `
		INSERT INTO assessment.item_skill_mappings
			(question_id, topic_id, clo_code, relevance, cognitive_level, generation_method,
			 authored_by, reviewed_by, confirmed_at, version, is_current, replaces_mapping_id)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, TRUE, $11)
		RETURNING id
	`, p.QuestionID, p.TopicID, p.CloCode, p.Relevance, p.CognitiveLevel, generationMethod,
		p.UserID, reviewedByParam, confirmedAtParam, newVersion, prevID,
	).Scan(&id)
	if err != nil {
		return nil, fmt.Errorf("insert item-skill mapping: %w", err)
	}

	action := "created"
	if prevID != nil {
		action = "superseded"
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO assessment.item_skill_mapping_review_history (mapping_id, action, new_values, reviewed_by)
		VALUES ($1, $2, jsonb_build_object('relevance', $3::float8, 'generationMethod', $4::text, 'version', $5::int), $6)
	`, id, action, p.Relevance, generationMethod, newVersion, p.UserID); err != nil {
		return nil, fmt.Errorf("record item-skill mapping review history: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit add item-skill mapping: %w", err)
	}

	return &dto.ItemSkillMapping{
		ID:               id.String(),
		QuestionID:       p.QuestionID.String(),
		TopicID:          p.TopicID.String(),
		TopicTitle:       topicTitle,
		CloCode:          p.CloCode,
		Relevance:        p.Relevance,
		CognitiveLevel:   p.CognitiveLevel,
		GenerationMethod: generationMethod,
		Version:          newVersion,
		IsCurrent:        true,
		IsValidated:      validated,
	}, nil
}

// ListItemSkillMappings returns the current Q-matrix rows for one
// question (every skill it's mapped to).
func (r *Repository) ListItemSkillMappings(ctx context.Context, questionID uuid.UUID) ([]dto.ItemSkillMapping, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT m.id, m.question_id, m.topic_id, t.title_en, m.clo_code, m.relevance, m.cognitive_level,
		       m.difficulty, m.discrimination, m.qmatrix_confidence, m.generation_method, m.version,
		       m.is_current, (m.confirmed_at IS NOT NULL)
		FROM assessment.item_skill_mappings m
		JOIN curriculum.topics t ON t.id = m.topic_id
		WHERE m.question_id = $1 AND m.is_current
		ORDER BY m.relevance DESC
	`, questionID)
	if err != nil {
		return nil, fmt.Errorf("list item-skill mappings: %w", err)
	}
	defer rows.Close()

	out := make([]dto.ItemSkillMapping, 0)
	for rows.Next() {
		var m dto.ItemSkillMapping
		if err := rows.Scan(
			&m.ID, &m.QuestionID, &m.TopicID, &m.TopicTitle, &m.CloCode, &m.Relevance, &m.CognitiveLevel,
			&m.Difficulty, &m.Discrimination, &m.QMatrixConfidence, &m.GenerationMethod, &m.Version,
			&m.IsCurrent, &m.IsValidated,
		); err != nil {
			return nil, fmt.Errorf("scan item-skill mapping: %w", err)
		}
		out = append(out, m)
	}
	return out, rows.Err()
}

// SyncItemSkillMappingToNeo4j mirrors one Q-matrix entry as
// (:Question {id})-[:ASSESSES {relevance, isValidated}]->(:Topic {id}).
// Best-effort, same contract as SyncPrerequisiteToNeo4j: Postgres commit
// is the success criterion.
func (r *Repository) SyncItemSkillMappingToNeo4j(ctx context.Context, mappingID, questionID, topicID uuid.UUID, relevance float64, isValidated bool) error {
	session := r.neo4j.NewSession(ctx, neo4jdriver.SessionConfig{AccessMode: neo4jdriver.AccessModeWrite})
	defer session.Close(ctx)

	_, err := session.Run(ctx, `
		MERGE (q:Question {id: $questionId})
		MERGE (t:Topic {id: $topicId})
		MERGE (q)-[rel:ASSESSES]->(t)
		SET rel.relevance = $relevance, rel.isValidated = $isValidated, rel.mappingId = $mappingId
	`, map[string]any{
		"questionId":  questionID.String(),
		"topicId":     topicID.String(),
		"relevance":   relevance,
		"isValidated": isValidated,
		"mappingId":   mappingID.String(),
	})
	if err != nil {
		return fmt.Errorf("sync item-skill mapping to neo4j: %w", err)
	}
	return nil
}

// MarkItemSkillMappingSynced flips neo4j_written = true after a
// successful SyncItemSkillMappingToNeo4j call.
func (r *Repository) MarkItemSkillMappingSynced(ctx context.Context, mappingID uuid.UUID) error {
	_, err := r.pool.Exec(ctx, `UPDATE assessment.item_skill_mappings SET neo4j_written = true WHERE id = $1`, mappingID)
	if err != nil {
		return fmt.Errorf("mark item-skill mapping synced: %w", err)
	}
	return nil
}

// ListUnsyncedItemSkillMappings mirrors ListUnsyncedPrerequisites for the
// Q-matrix's bulk resync endpoint.
func (r *Repository) ListUnsyncedItemSkillMappings(ctx context.Context) ([]UnsyncedItemSkillMapping, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, question_id, topic_id, relevance, (confirmed_at IS NOT NULL)
		FROM assessment.item_skill_mappings
		WHERE NOT neo4j_written AND is_current
	`)
	if err != nil {
		return nil, fmt.Errorf("list unsynced item-skill mappings: %w", err)
	}
	defer rows.Close()

	out := make([]UnsyncedItemSkillMapping, 0)
	for rows.Next() {
		var u UnsyncedItemSkillMapping
		if err := rows.Scan(&u.ID, &u.QuestionID, &u.TopicID, &u.Relevance, &u.IsValidated); err != nil {
			return nil, fmt.Errorf("scan unsynced item-skill mapping: %w", err)
		}
		out = append(out, u)
	}
	return out, rows.Err()
}
