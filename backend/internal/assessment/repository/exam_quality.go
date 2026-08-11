package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// Capability 4B: read side pulls everything the quality computation
// needs in three queries; the metrics themselves are computed in the
// service (small result sets -- one exam's questions and answers).

type ExamForQuality struct {
	ID          uuid.UUID
	SchoolID    uuid.UUID
	SubjectCode string
	GradeLevel  int
	Title       string
	Status      string
}

func (r *Repository) FetchExamForQuality(ctx context.Context, examID uuid.UUID) (*ExamForQuality, error) {
	var e ExamForQuality
	err := r.pool.QueryRow(ctx, `
		SELECT id, school_id, subject_code, grade_level, title, status
		FROM assessment.exams WHERE id = $1
	`, examID).Scan(&e.ID, &e.SchoolID, &e.SubjectCode, &e.GradeLevel, &e.Title, &e.Status)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("fetch exam for quality: %w", err)
	}
	return &e, nil
}

type QuestionForQuality struct {
	ID               uuid.UUID
	SequenceNumber   int
	QuestionType     string
	Marks            int
	StatedDifficulty *string
	CLOCode          *string
	TopicID          *uuid.UUID
}

func (r *Repository) FetchQuestionsForQuality(ctx context.Context, examID uuid.UUID) ([]QuestionForQuality, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, sequence_number, question_type, marks, difficulty_level, clo_code, topic_id
		FROM assessment.questions
		WHERE exam_id = $1
		ORDER BY sequence_number
	`, examID)
	if err != nil {
		return nil, fmt.Errorf("fetch questions for quality: %w", err)
	}
	defer rows.Close()

	var out []QuestionForQuality
	for rows.Next() {
		var q QuestionForQuality
		if err := rows.Scan(&q.ID, &q.SequenceNumber, &q.QuestionType, &q.Marks,
			&q.StatedDifficulty, &q.CLOCode, &q.TopicID); err != nil {
			return nil, fmt.Errorf("scan question for quality: %w", err)
		}
		out = append(out, q)
	}
	return out, rows.Err()
}

type AnswerForQuality struct {
	QuestionID    uuid.UUID
	StudentID     uuid.UUID
	ScoreRatio    float64 // marks_awarded / marks_possible
	Passed        bool
	TimeSpentSecs *int
	// The student's overall attempt percentage -- the classic
	// discrimination index splits on this.
	AttemptPct *float64
	// mastery_records confidence on this question's topic; nil = no
	// evidence (which is NOT weakness -- such students are excluded from
	// the mastery split, mirroring 3A's rule). NOTE: 3A refreshes
	// mastery FROM this exam's results, so the split is partly
	// self-referential -- acceptable until pre-exam mastery snapshots exist.
	TopicMastery *float64
}

func (r *Repository) FetchAnswersForQuality(ctx context.Context, examID uuid.UUID) ([]AnswerForQuality, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT sa.question_id, sa.student_id,
		       (CASE WHEN sa.marks_possible > 0 THEN sa.marks_awarded / sa.marks_possible ELSE 0 END)::float8,
		       COALESCE(sa.passed, sa.marks_awarded >= 0.5 * sa.marks_possible),
		       sa.time_spent_secs,
		       a.percentage::float8,
		       mr.confidence::float8
		FROM assessment.student_answers sa
		JOIN assessment.exam_attempts a ON a.id = sa.attempt_id
		JOIN assessment.questions q ON q.id = sa.question_id
		LEFT JOIN students.mastery_records mr
		       ON mr.student_id = sa.student_id AND mr.topic_id = q.topic_id
		WHERE q.exam_id = $1 AND sa.marks_awarded IS NOT NULL
	`, examID)
	if err != nil {
		return nil, fmt.Errorf("fetch answers for quality: %w", err)
	}
	defer rows.Close()

	var out []AnswerForQuality
	for rows.Next() {
		var a AnswerForQuality
		if err := rows.Scan(&a.QuestionID, &a.StudentID, &a.ScoreRatio, &a.Passed,
			&a.TimeSpentSecs, &a.AttemptPct, &a.TopicMastery); err != nil {
			return nil, fmt.Errorf("scan answer for quality: %w", err)
		}
		out = append(out, a)
	}
	return out, rows.Err()
}

func (r *Repository) FinalizedAttemptCount(ctx context.Context, examID uuid.UUID) (int, error) {
	var n int
	err := r.pool.QueryRow(ctx, `
		SELECT count(*) FROM assessment.exam_attempts
		WHERE exam_id = $1 AND total_score IS NOT NULL
	`, examID).Scan(&n)
	if err != nil {
		return 0, fmt.Errorf("count finalized attempts: %w", err)
	}
	return n, nil
}

type MandatoryCLO struct {
	Code        string
	Description string
}

func (r *Repository) FetchMandatoryCLOs(ctx context.Context, subjectCode string, gradeLevel int) ([]MandatoryCLO, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT code, description_en FROM curriculum.clos
		WHERE subject_code = $1 AND grade_level = $2 AND is_mandatory
		ORDER BY code
	`, subjectCode, gradeLevel)
	if err != nil {
		return nil, fmt.Errorf("fetch mandatory clos: %w", err)
	}
	defer rows.Close()

	var out []MandatoryCLO
	for rows.Next() {
		var c MandatoryCLO
		if err := rows.Scan(&c.Code, &c.Description); err != nil {
			return nil, fmt.Errorf("scan mandatory clo: %w", err)
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

// WriteCalibratedDifficulties is the "recalibrates for future exams"
// half of difficulty calibration: performance-derived difficulty lands in
// its own column, the stated difficulty_level is never touched.
func (r *Repository) WriteCalibratedDifficulties(ctx context.Context, calibrated map[uuid.UUID]string) error {
	if len(calibrated) == 0 {
		return nil
	}
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin calibration tx: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	for id, diff := range calibrated {
		if _, err := tx.Exec(ctx,
			`UPDATE assessment.questions SET calibrated_difficulty = $2 WHERE id = $1`, id, diff); err != nil {
			return fmt.Errorf("write calibrated difficulty for question %s: %w", id, err)
		}
	}
	return tx.Commit(ctx)
}

// UpsertExamQualityReport persists the report (recompute-on-read, so the
// row is always the latest run). Scalars are what 4C aggregates.
func (r *Repository) UpsertExamQualityReport(
	ctx context.Context,
	examID, schoolID uuid.UUID,
	attemptsAnalyzed int,
	discriminationAvg *float64,
	qualityScore float64,
	reportJSON []byte,
) error {
	_, err := r.pool.Exec(ctx, `
		INSERT INTO assessment.exam_quality_reports
			(exam_id, school_id, attempts_analyzed, discrimination_avg, quality_score, report, computed_at)
		VALUES ($1, $2, $3, $4, $5, $6, now())
		ON CONFLICT (exam_id) DO UPDATE SET
			attempts_analyzed = EXCLUDED.attempts_analyzed,
			discrimination_avg = EXCLUDED.discrimination_avg,
			quality_score = EXCLUDED.quality_score,
			report = EXCLUDED.report,
			computed_at = now()
	`, examID, schoolID, attemptsAnalyzed, discriminationAvg, qualityScore, reportJSON)
	if err != nil {
		return fmt.Errorf("upsert exam quality report: %w", err)
	}
	return nil
}
