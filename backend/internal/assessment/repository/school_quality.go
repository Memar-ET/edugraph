package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// Capability 4C: read-side component queries + persistence for the
// composite school quality score.

type SchoolSubjectGrade struct {
	SchoolID    uuid.UUID
	SubjectCode string
	GradeLevel  int
}

// QualityCombos enumerates every (school, subject, grade) with at least
// one published exam -- the nightly worker's work list. A school that
// never published an exam has no assessment signal to score.
func (r *Repository) QualityCombos(ctx context.Context) ([]SchoolSubjectGrade, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT DISTINCT school_id, subject_code, grade_level
		FROM assessment.exams WHERE status = 'published'
	`)
	if err != nil {
		return nil, fmt.Errorf("list quality combos: %w", err)
	}
	defer rows.Close()

	var out []SchoolSubjectGrade
	for rows.Next() {
		var c SchoolSubjectGrade
		if err := rows.Scan(&c.SchoolID, &c.SubjectCode, &c.GradeLevel); err != nil {
			return nil, fmt.Errorf("scan quality combo: %w", err)
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

// PublishedExamsMissingQuality: published exams in the combo without a
// quality report yet -- the worker fills these in before averaging.
func (r *Repository) PublishedExamsMissingQuality(ctx context.Context, c SchoolSubjectGrade) ([]uuid.UUID, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT e.id
		FROM assessment.exams e
		LEFT JOIN assessment.exam_quality_reports r ON r.exam_id = e.id
		WHERE e.school_id = $1 AND e.subject_code = $2 AND e.grade_level = $3
		  AND e.status = 'published' AND r.id IS NULL
	`, c.SchoolID, c.SubjectCode, c.GradeLevel)
	if err != nil {
		return nil, fmt.Errorf("list exams missing quality: %w", err)
	}
	defer rows.Close()

	var out []uuid.UUID
	for rows.Next() {
		var id uuid.UUID
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("scan exam missing quality: %w", err)
		}
		out = append(out, id)
	}
	return out, rows.Err()
}

// CLOCoverageStats: how many of the subject+grade's MANDATORY CLOs are
// tested by at least one question in this school's published exams.
func (r *Repository) CLOCoverageStats(ctx context.Context, c SchoolSubjectGrade) (total, tested int, err error) {
	err = r.pool.QueryRow(ctx, `
		SELECT
			(SELECT count(*) FROM curriculum.clos cl
			 WHERE cl.subject_code = $2 AND cl.grade_level = $3 AND cl.is_mandatory),
			(SELECT count(DISTINCT q.clo_code)
			 FROM assessment.questions q
			 JOIN assessment.exams e ON e.id = q.exam_id
			 JOIN curriculum.clos cl ON cl.code = q.clo_code
			 WHERE e.school_id = $1 AND e.subject_code = $2 AND e.grade_level = $3
			   AND e.status = 'published' AND cl.is_mandatory)
	`, c.SchoolID, c.SubjectCode, c.GradeLevel).Scan(&total, &tested)
	if err != nil {
		return 0, 0, fmt.Errorf("clo coverage stats: %w", err)
	}
	return total, tested, nil
}

// StudentMasteryStats: of the school+grade cohort, how many students
// have average OPEN gap severity below 0.3 in this subject (students
// with no open gaps count as mastered -- absence of detected gaps is the
// healthy state, unlike 3A's per-prerequisite "no evidence" rule).
func (r *Repository) StudentMasteryStats(ctx context.Context, c SchoolSubjectGrade) (total, mastered int, err error) {
	err = r.pool.QueryRow(ctx, `
		SELECT count(*), count(*) FILTER (WHERE avg_sev < 0.3)
		FROM (
			SELECT s.id, COALESCE(avg(g.severity_score), 0) AS avg_sev
			FROM students s
			LEFT JOIN students.gap_records g
			       ON g.student_id = s.id
			      AND g.resolved_at IS NULL
			      AND g.topic_id IN (SELECT t.id FROM curriculum.topics t WHERE t.subject_code = $2)
			WHERE s.school_id = $1 AND s.grade_level = $3
			GROUP BY s.id
		) x
	`, c.SchoolID, c.SubjectCode, c.GradeLevel).Scan(&total, &mastered)
	if err != nil {
		return 0, 0, fmt.Errorf("student mastery stats: %w", err)
	}
	return total, mastered, nil
}

// ExamDiscriminationAvg: mean discrimination index (-1..1) across the
// combo's quality reports. nil when no report has a computable index.
func (r *Repository) ExamDiscriminationAvg(ctx context.Context, c SchoolSubjectGrade) (*float64, error) {
	var avg *float64
	err := r.pool.QueryRow(ctx, `
		SELECT avg(r.discrimination_avg)::float8
		FROM assessment.exam_quality_reports r
		JOIN assessment.exams e ON e.id = r.exam_id
		WHERE e.school_id = $1 AND e.subject_code = $2 AND e.grade_level = $3
	`, c.SchoolID, c.SubjectCode, c.GradeLevel).Scan(&avg)
	if err != nil {
		return nil, fmt.Errorf("exam discrimination avg: %w", err)
	}
	return avg, nil
}

// ComplianceStats: % of the subject+grade's topics assessed by at least
// one published-exam question (directly via topic_id, or through the
// question's CLO's best topic mapping -- same resolution rule as 3A).
func (r *Repository) ComplianceStats(ctx context.Context, c SchoolSubjectGrade) (totalTopics, assessed int, err error) {
	err = r.pool.QueryRow(ctx, `
		SELECT
			(SELECT count(*) FROM curriculum.topics t
			 WHERE t.subject_code = $2 AND t.grade_level = $3),
			(SELECT count(DISTINCT t.id)
			 FROM assessment.questions q
			 JOIN assessment.exams e ON e.id = q.exam_id
			 LEFT JOIN LATERAL (
				SELECT m.topic_id FROM curriculum.topic_clo_mappings m
				WHERE m.clo_code = q.clo_code
				ORDER BY m.alignment_score DESC NULLS LAST LIMIT 1
			 ) tcm ON q.topic_id IS NULL
			 JOIN curriculum.topics t ON t.id = COALESCE(q.topic_id, tcm.topic_id)
			 WHERE e.school_id = $1 AND e.subject_code = $2 AND e.grade_level = $3
			   AND e.status = 'published'
			   AND t.subject_code = $2 AND t.grade_level = $3)
	`, c.SchoolID, c.SubjectCode, c.GradeLevel).Scan(&totalTopics, &assessed)
	if err != nil {
		return 0, 0, fmt.Errorf("compliance stats: %w", err)
	}
	return totalTopics, assessed, nil
}

type QualityScoreRow struct {
	SubjectCode          string
	GradeLevel           int
	CLOCoveragePct       float64
	StudentMasteryPct    float64
	ExamQualityAvg       float64
	CurriculumCompliance float64
	CompositeScore       float64
	FlaggedForReview     bool
	ComputedAt           time.Time
}

// UpsertQualityScore persists one combo's score and reports whether this
// write TRANSITIONED the combo into the flagged state -- the caller only
// notifies the Ministry on false->true so nightly recomputes don't spam.
func (r *Repository) UpsertQualityScore(ctx context.Context, c SchoolSubjectGrade, row QualityScoreRow) (newlyFlagged bool, err error) {
	wasFlagged := false
	err = r.pool.QueryRow(ctx, `
		SELECT flagged_for_review FROM schools.quality_scores
		WHERE school_id = $1 AND subject_code = $2 AND grade_level = $3
	`, c.SchoolID, c.SubjectCode, c.GradeLevel).Scan(&wasFlagged)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return false, fmt.Errorf("read prior quality flag: %w", err)
	}

	_, err = r.pool.Exec(ctx, `
		INSERT INTO schools.quality_scores
			(school_id, subject_code, grade_level, clo_coverage_pct, student_mastery_pct,
			 exam_quality_avg, curriculum_compliance, composite_score, flagged_for_review, computed_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, now())
		ON CONFLICT (school_id, subject_code, grade_level) DO UPDATE SET
			clo_coverage_pct = EXCLUDED.clo_coverage_pct,
			student_mastery_pct = EXCLUDED.student_mastery_pct,
			exam_quality_avg = EXCLUDED.exam_quality_avg,
			curriculum_compliance = EXCLUDED.curriculum_compliance,
			composite_score = EXCLUDED.composite_score,
			flagged_for_review = EXCLUDED.flagged_for_review,
			computed_at = now()
	`, c.SchoolID, c.SubjectCode, c.GradeLevel,
		row.CLOCoveragePct, row.StudentMasteryPct, row.ExamQualityAvg,
		row.CurriculumCompliance, row.CompositeScore, row.FlaggedForReview)
	if err != nil {
		return false, fmt.Errorf("upsert quality score: %w", err)
	}

	return row.FlaggedForReview && !wasFlagged, nil
}

func (r *Repository) ListQualityScores(ctx context.Context, schoolID uuid.UUID) ([]QualityScoreRow, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT subject_code, grade_level, clo_coverage_pct::float8, student_mastery_pct::float8,
		       exam_quality_avg::float8, curriculum_compliance::float8, composite_score::float8,
		       flagged_for_review, computed_at
		FROM schools.quality_scores
		WHERE school_id = $1
		ORDER BY subject_code, grade_level
	`, schoolID)
	if err != nil {
		return nil, fmt.Errorf("list quality scores: %w", err)
	}
	defer rows.Close()

	var out []QualityScoreRow
	for rows.Next() {
		var q QualityScoreRow
		if err := rows.Scan(&q.SubjectCode, &q.GradeLevel, &q.CLOCoveragePct, &q.StudentMasteryPct,
			&q.ExamQualityAvg, &q.CurriculumCompliance, &q.CompositeScore,
			&q.FlaggedForReview, &q.ComputedAt); err != nil {
			return nil, fmt.Errorf("scan quality score: %w", err)
		}
		out = append(out, q)
	}
	return out, rows.Err()
}

// NotifyMinistryOfFlag fans a compliance warning out to every
// ministry_admin's notification feed (the existing per-user
// notifications table -- no new channel infra).
func (r *Repository) NotifyMinistryOfFlag(ctx context.Context, c SchoolSubjectGrade, compliance float64) error {
	var schoolName string
	if err := r.pool.QueryRow(ctx,
		`SELECT name FROM schools WHERE id = $1`, c.SchoolID).Scan(&schoolName); err != nil {
		schoolName = c.SchoolID.String()
	}
	_, err := r.pool.Exec(ctx, `
		INSERT INTO notifications (user_id, title, body)
		SELECT u.id,
		       'School flagged for curriculum compliance review',
		       format('%s: %s grade %s curriculum compliance is %s%% -- mandatory sections may be skipped. Review recommended.',
		              $1::text, $2::text, $3::text, $4::text)
		FROM users u WHERE u.role = 'ministry_admin'
	`, schoolName, c.SubjectCode, fmt.Sprint(c.GradeLevel), fmt.Sprintf("%.0f", compliance))
	if err != nil {
		return fmt.Errorf("notify ministry of flag: %w", err)
	}
	return nil
}
