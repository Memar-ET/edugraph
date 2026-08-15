package repository

import (
	"context"
	"fmt"

	"github.com/edugraph-ai/edugraph/internal/curriculum/dto"
)

// lowConfidenceThreshold: below this (or NULL/never-reviewed), a Q-matrix
// mapping or prerequisite edge is flagged for review. Not a value from
// the spec itself -- a documented, defensible threshold choice, same
// convention as WEAK_THRESHOLD on the ai-service side.
const lowConfidenceThreshold = 0.5

// ambiguousRelevanceGap: when a question has multiple current mappings
// and the gap between the highest and lowest relevance is smaller than
// this, no single skill clearly dominates -- "ambiguous."
const ambiguousRelevanceGap = 0.2

// QMatrixQuality answers "detect items with missing or low-confidence
// skill mappings" / "flag weak or ambiguous Q-matrix mappings" (EG-GCKT
// checklist sections 5/16) for every question in a subject.
func (r *Repository) QMatrixQuality(ctx context.Context, subjectCode string) (*dto.QMatrixQualityReport, error) {
	report := &dto.QMatrixQualityReport{SubjectCode: subjectCode}

	if err := r.pool.QueryRow(ctx, `
		SELECT count(*) FROM assessment.questions q
		JOIN assessment.exams e ON e.id = q.exam_id
		WHERE e.subject_code = $1
	`, subjectCode).Scan(&report.TotalQuestions); err != nil {
		return nil, fmt.Errorf("count subject questions: %w", err)
	}

	missingRows, err := r.pool.Query(ctx, `
		SELECT q.id, q.question_text, q.exam_id
		FROM assessment.questions q
		JOIN assessment.exams e ON e.id = q.exam_id
		WHERE e.subject_code = $1
		  AND NOT EXISTS (
		    SELECT 1 FROM assessment.item_skill_mappings m WHERE m.question_id = q.id AND m.is_current
		  )
	`, subjectCode)
	if err != nil {
		return nil, fmt.Errorf("query missing mappings: %w", err)
	}
	defer missingRows.Close()
	for missingRows.Next() {
		var issue dto.QuestionMappingIssue
		if err := missingRows.Scan(&issue.QuestionID, &issue.QuestionText, &issue.ExamID); err != nil {
			return nil, fmt.Errorf("scan missing mapping: %w", err)
		}
		issue.Reason = "no Q-matrix mapping exists for this question"
		report.MissingMappings = append(report.MissingMappings, issue)
	}
	if err := missingRows.Err(); err != nil {
		return nil, fmt.Errorf("iterate missing mappings: %w", err)
	}

	lowConfRows, err := r.pool.Query(ctx, `
		SELECT DISTINCT q.id, q.question_text, q.exam_id
		FROM assessment.questions q
		JOIN assessment.exams e ON e.id = q.exam_id
		JOIN assessment.item_skill_mappings m ON m.question_id = q.id AND m.is_current
		WHERE e.subject_code = $1
		  AND (m.qmatrix_confidence IS NULL OR m.qmatrix_confidence < $2)
	`, subjectCode, lowConfidenceThreshold)
	if err != nil {
		return nil, fmt.Errorf("query low-confidence mappings: %w", err)
	}
	defer lowConfRows.Close()
	for lowConfRows.Next() {
		var issue dto.QuestionMappingIssue
		if err := lowConfRows.Scan(&issue.QuestionID, &issue.QuestionText, &issue.ExamID); err != nil {
			return nil, fmt.Errorf("scan low-confidence mapping: %w", err)
		}
		issue.Reason = "Q-matrix confidence is missing or below threshold"
		report.LowConfidenceMappings = append(report.LowConfidenceMappings, issue)
	}
	if err := lowConfRows.Err(); err != nil {
		return nil, fmt.Errorf("iterate low-confidence mappings: %w", err)
	}

	ambiguousRows, err := r.pool.Query(ctx, `
		SELECT q.id, q.question_text, q.exam_id,
		       array_agg(t.title_en ORDER BY m.relevance DESC) AS titles,
		       max(m.relevance) - min(m.relevance) AS spread
		FROM assessment.questions q
		JOIN assessment.exams e ON e.id = q.exam_id
		JOIN assessment.item_skill_mappings m ON m.question_id = q.id AND m.is_current
		JOIN curriculum.topics t ON t.id = m.topic_id
		WHERE e.subject_code = $1
		GROUP BY q.id, q.question_text, q.exam_id
		HAVING count(*) > 1 AND max(m.relevance) - min(m.relevance) < $2
	`, subjectCode, ambiguousRelevanceGap)
	if err != nil {
		return nil, fmt.Errorf("query ambiguous mappings: %w", err)
	}
	defer ambiguousRows.Close()
	for ambiguousRows.Next() {
		var issue dto.QuestionMappingIssue
		var spread float64
		if err := ambiguousRows.Scan(&issue.QuestionID, &issue.QuestionText, &issue.ExamID, &issue.TopicTitles, &spread); err != nil {
			return nil, fmt.Errorf("scan ambiguous mapping: %w", err)
		}
		issue.Reason = "multiple mapped skills with no clearly dominant relevance"
		report.AmbiguousMappings = append(report.AmbiguousMappings, issue)
	}
	if err := ambiguousRows.Err(); err != nil {
		return nil, fmt.Errorf("iterate ambiguous mappings: %w", err)
	}

	return report, nil
}

// PrerequisiteQuality answers "run structural validation for orphaned
// skills... and duplicate edges" (EG-GCKT checklist section 4) for every
// topic in a subject. Duplicate edges are already impossible at the DB
// constraint level (UNIQUE(topic_id, prerequisite_id, edge_type), V033),
// so this focuses on isolation and low-confidence edges.
func (r *Repository) PrerequisiteQuality(ctx context.Context, subjectCode string) (*dto.PrerequisiteQualityReport, error) {
	report := &dto.PrerequisiteQualityReport{SubjectCode: subjectCode}

	if err := r.pool.QueryRow(ctx, `SELECT count(*) FROM curriculum.topics WHERE subject_code = $1`, subjectCode).
		Scan(&report.TotalTopics); err != nil {
		return nil, fmt.Errorf("count subject topics: %w", err)
	}

	orphanRows, err := r.pool.Query(ctx, `
		SELECT t.id, t.title_en
		FROM curriculum.topics t
		WHERE t.subject_code = $1
		  AND NOT EXISTS (SELECT 1 FROM curriculum.topic_prerequisites tp WHERE tp.topic_id = t.id)
		  AND NOT EXISTS (SELECT 1 FROM curriculum.topic_prerequisites tp WHERE tp.prerequisite_id = t.id)
	`, subjectCode)
	if err != nil {
		return nil, fmt.Errorf("query orphaned topics: %w", err)
	}
	defer orphanRows.Close()
	for orphanRows.Next() {
		var o dto.OrphanedTopic
		if err := orphanRows.Scan(&o.TopicID, &o.Title); err != nil {
			return nil, fmt.Errorf("scan orphaned topic: %w", err)
		}
		report.OrphanedTopics = append(report.OrphanedTopics, o)
	}
	if err := orphanRows.Err(); err != nil {
		return nil, fmt.Errorf("iterate orphaned topics: %w", err)
	}

	edgeRows, err := r.pool.Query(ctx, `
		SELECT tp.topic_id, t.title_en, tp.prerequisite_id, p.title_en,
		       tp.edge_type, tp.confidence, (tp.confirmed_at IS NOT NULL)
		FROM curriculum.topic_prerequisites tp
		JOIN curriculum.topics t ON t.id = tp.topic_id
		JOIN curriculum.topics p ON p.id = tp.prerequisite_id
		WHERE t.subject_code = $1 AND (tp.confidence IS NULL OR tp.confidence < $2)
	`, subjectCode, lowConfidenceThreshold)
	if err != nil {
		return nil, fmt.Errorf("query low-confidence edges: %w", err)
	}
	defer edgeRows.Close()
	for edgeRows.Next() {
		var e dto.LowConfidenceEdgeIssue
		if err := edgeRows.Scan(
			&e.TopicID, &e.TopicTitle, &e.PrerequisiteTopicID, &e.PrerequisiteTitle,
			&e.EdgeType, &e.Confidence, &e.IsValidated,
		); err != nil {
			return nil, fmt.Errorf("scan low-confidence edge: %w", err)
		}
		report.LowConfidenceEdges = append(report.LowConfidenceEdges, e)
	}
	if err := edgeRows.Err(); err != nil {
		return nil, fmt.Errorf("iterate low-confidence edges: %w", err)
	}

	return report, nil
}
