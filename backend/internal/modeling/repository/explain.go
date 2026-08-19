package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/edugraph-ai/edugraph/internal/modeling/dto"
)

var (
	ErrSkillStateNotFound = errors.New("no skill state for this student/topic yet")
	ErrStudentNotFound    = errors.New("student not found")
)

// AuthContext resolves everything Service.Explain needs to authorize a
// caller against an arbitrary student id in the URL -- role, the
// caller's own students.id (nil unless the caller IS a student), and the
// caller's school (users.school_id, set for teacher/school_admin
// accounts). See this codebase's documented career-matches IDOR fix
// (checklist 11.3) for why a bare {id} path param must never be trusted
// without exactly this kind of server-side ownership check.
type AuthContext struct {
	Role         string
	OwnStudentID *uuid.UUID
	CallerSchool *uuid.UUID
}

func (r *Repository) FetchAuthContext(ctx context.Context, userID uuid.UUID) (*AuthContext, error) {
	var a AuthContext
	err := r.pool.QueryRow(ctx, `
		SELECT u.role, s.id, u.school_id
		FROM users u
		LEFT JOIN students s ON s.user_id = u.id
		WHERE u.id = $1
	`, userID).Scan(&a.Role, &a.OwnStudentID, &a.CallerSchool)
	if err != nil {
		return nil, fmt.Errorf("fetch auth context: %w", err)
	}
	return &a, nil
}

// FetchStudentSchool resolves the target student's school for the
// teacher/school_admin same-school authorization check.
func (r *Repository) FetchStudentSchool(ctx context.Context, studentID uuid.UUID) (uuid.UUID, error) {
	var schoolID uuid.UUID
	err := r.pool.QueryRow(ctx, `SELECT school_id FROM students WHERE id = $1`, studentID).Scan(&schoolID)
	if errors.Is(err, pgx.ErrNoRows) {
		return uuid.Nil, ErrStudentNotFound
	}
	if err != nil {
		return uuid.Nil, fmt.Errorf("fetch student school: %w", err)
	}
	return schoolID, nil
}

// SkillState is the raw students.skill_states row this package's
// explain/next-best-action reads need -- kept separate from
// dto.ExplanationCurrentState so the repository layer isn't coupled to
// exactly one caller's JSON shape.
type SkillState struct {
	MasteryProbability *float64
	MasteryStatus      string
	Trend              *string
	EvidenceCount      int
	ForgettingRisk     *float64
	LastSeen           *time.Time
}

// FetchSkillState returns the current fused EG-GCKT state for one
// (student, topic) pair. ErrSkillStateNotFound is the correct, expected
// response for a pair with no evidence yet (cold start via row absence,
// Milestone 0) -- callers must not treat it as a server error.
func (r *Repository) FetchSkillState(ctx context.Context, studentID, topicID uuid.UUID) (*SkillState, string, error) {
	var s SkillState
	var topicTitle string
	err := r.pool.QueryRow(ctx, `
		SELECT ss.mastery_probability, ss.mastery_status, ss.trend, ss.evidence_count,
		       ss.forgetting_risk, ss.last_seen, t.title_en
		FROM students.skill_states ss
		JOIN curriculum.topics t ON t.id = ss.topic_id
		WHERE ss.student_id = $1 AND ss.topic_id = $2
	`, studentID, topicID).Scan(
		&s.MasteryProbability, &s.MasteryStatus, &s.Trend, &s.EvidenceCount,
		&s.ForgettingRisk, &s.LastSeen, &topicTitle,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, "", ErrSkillStateNotFound
	}
	if err != nil {
		return nil, "", fmt.Errorf("fetch skill state: %w", err)
	}
	return &s, topicTitle, nil
}

// FetchRecentEvidence returns the most recent modeling.evidence_log row
// per distinct provenance for (student, topic) -- one representative
// piece of evidence per contributing engine, not every row ever written.
func (r *Repository) FetchRecentEvidence(ctx context.Context, studentID, topicID uuid.UUID) ([]dto.ExplanationEvidenceItem, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT DISTINCT ON (provenance) provenance, estimate, uncertainty, reliability, created_at
		FROM modeling.evidence_log
		WHERE student_id = $1 AND topic_id = $2
		ORDER BY provenance, created_at DESC
	`, studentID, topicID)
	if err != nil {
		return nil, fmt.Errorf("fetch recent evidence: %w", err)
	}
	defer rows.Close()

	out := make([]dto.ExplanationEvidenceItem, 0)
	for rows.Next() {
		var e dto.ExplanationEvidenceItem
		if err := rows.Scan(&e.Provenance, &e.Estimate, &e.Uncertainty, &e.Reliability, &e.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan evidence item: %w", err)
		}
		out = append(out, e)
	}
	return out, rows.Err()
}

// FetchPrerequisiteMastery returns the topic's direct prerequisites (any
// edge type) with the student's current mastery on each -- the
// structural-context half of the explanation.
func (r *Repository) FetchPrerequisiteMastery(ctx context.Context, studentID, topicID uuid.UUID) ([]dto.PrerequisiteMasterySummary, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT p.id, p.title_en, tp.edge_type, ss.mastery_probability
		FROM curriculum.topic_prerequisites tp
		JOIN curriculum.topics p ON p.id = tp.prerequisite_id
		LEFT JOIN students.skill_states ss ON ss.student_id = $1 AND ss.topic_id = p.id
		WHERE tp.topic_id = $2
		ORDER BY tp.edge_type, tp.weight DESC
	`, studentID, topicID)
	if err != nil {
		return nil, fmt.Errorf("fetch prerequisite mastery: %w", err)
	}
	defer rows.Close()

	out := make([]dto.PrerequisiteMasterySummary, 0)
	for rows.Next() {
		var p dto.PrerequisiteMasterySummary
		if err := rows.Scan(&p.TopicID, &p.Title, &p.EdgeType, &p.MasteryProbability); err != nil {
			return nil, fmt.Errorf("scan prerequisite mastery: %w", err)
		}
		out = append(out, p)
	}
	return out, rows.Err()
}
