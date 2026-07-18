package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/edugraph-ai/edugraph/internal/assessment/dto"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// FetchInsightForStudentExam reads the Capability 3A Exam Insight layer
// for one (student, exam) -- written by the ai-service gap worker after
// the attempt is fully graded. ErrNotFound covers both "never attempted"
// and "graded but not analyzed yet" (the worker may still be running);
// callers surface that as a retryable "not ready" rather than a failure.
func (r *Repository) FetchInsightForStudentExam(ctx context.Context, studentID, examID uuid.UUID) (*dto.ExamInsight, error) {
	const q = `
		SELECT attempt_id, exam_id, total_score, percentage, passed,
		       gaps_found, llm_exam_summary, llm_model, generated_at
		FROM students.exam_insights
		WHERE student_id = $1 AND exam_id = $2
	`
	var out dto.ExamInsight
	err := r.pool.QueryRow(ctx, q, studentID, examID).Scan(
		&out.AttemptID, &out.ExamID, &out.TotalScore, &out.Percentage, &out.Passed,
		&out.GapsFound, &out.Summary, &out.LlmModel, &out.GeneratedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("fetch exam insight: %w", err)
	}

	gaps, err := r.fetchGapRecords(ctx, out.AttemptID)
	if err != nil {
		return nil, err
	}
	out.Gaps = gaps
	return &out, nil
}

// fetchGapRecords loads the Granular Layer for one attempt, worst first,
// with topic titles resolved for both the symptom and the root cause.
func (r *Repository) fetchGapRecords(ctx context.Context, attemptID uuid.UUID) ([]dto.GapRecordEntry, error) {
	const q = `
		SELECT g.question_id, g.topic_id, st.title_en, g.clo_code,
		       g.root_cause_topic_id, rt.title_en, rt.grade_level,
		       g.severity_score, g.prerequisite_depth, g.llm_explanation
		FROM students.gap_records g
		JOIN curriculum.topics st ON st.id = g.topic_id
		LEFT JOIN curriculum.topics rt ON rt.id = g.root_cause_topic_id
		WHERE g.attempt_id = $1
		ORDER BY g.severity_score DESC
	`
	rows, err := r.pool.Query(ctx, q, attemptID)
	if err != nil {
		return nil, fmt.Errorf("fetch gap records: %w", err)
	}
	defer rows.Close()

	gaps := make([]dto.GapRecordEntry, 0)
	for rows.Next() {
		var g dto.GapRecordEntry
		if err := rows.Scan(
			&g.QuestionID, &g.SymptomTopicID, &g.SymptomTopicTitle, &g.CloCode,
			&g.RootCauseTopicID, &g.RootCauseTopicTitle, &g.RootCauseGrade,
			&g.SeverityScore, &g.PrerequisiteDepth, &g.LlmExplanation,
		); err != nil {
			return nil, fmt.Errorf("scan gap record: %w", err)
		}
		gaps = append(gaps, g)
	}
	return gaps, rows.Err()
}

// FetchInsightsForExam backs the teacher-facing per-exam list -- one
// summary row per analyzed attempt, no gap detail.
func (r *Repository) FetchInsightsForExam(ctx context.Context, examID uuid.UUID) ([]dto.ExamInsightListEntry, error) {
	const q = `
		SELECT student_id, attempt_id, total_score, percentage, passed,
		       gaps_found, llm_exam_summary, generated_at
		FROM students.exam_insights
		WHERE exam_id = $1
		ORDER BY generated_at DESC
	`
	rows, err := r.pool.Query(ctx, q, examID)
	if err != nil {
		return nil, fmt.Errorf("fetch exam insights: %w", err)
	}
	defer rows.Close()

	out := make([]dto.ExamInsightListEntry, 0)
	for rows.Next() {
		var e dto.ExamInsightListEntry
		if err := rows.Scan(
			&e.StudentID, &e.AttemptID, &e.TotalScore, &e.Percentage, &e.Passed,
			&e.GapsFound, &e.Summary, &e.GeneratedAt,
		); err != nil {
			return nil, fmt.Errorf("scan exam insight entry: %w", err)
		}
		out = append(out, e)
	}
	return out, rows.Err()
}

// FetchSubjectProfiles reads the Subject Health Layer for the student
// dashboard ("Physics: 75% healthy"), joined to subject names.
func (r *Repository) FetchSubjectProfiles(ctx context.Context, studentID uuid.UUID) ([]dto.SubjectProfile, error) {
	const q = `
		SELECT p.subject_code, s.name_en, p.grade_level, p.current_mastery_pct,
		       p.top_weak_areas, p.exams_analyzed, p.last_updated
		FROM students.subject_profiles p
		JOIN curriculum.subjects s ON s.code = p.subject_code
		WHERE p.student_id = $1
		ORDER BY s.name_en
	`
	rows, err := r.pool.Query(ctx, q, studentID)
	if err != nil {
		return nil, fmt.Errorf("fetch subject profiles: %w", err)
	}
	defer rows.Close()

	out := make([]dto.SubjectProfile, 0)
	for rows.Next() {
		var p dto.SubjectProfile
		var weakAreas []byte
		if err := rows.Scan(
			&p.SubjectCode, &p.SubjectName, &p.GradeLevel, &p.CurrentMasteryPct,
			&weakAreas, &p.ExamsAnalyzed, &p.LastUpdated,
		); err != nil {
			return nil, fmt.Errorf("scan subject profile: %w", err)
		}
		p.TopWeakAreas = weakAreas
		out = append(out, p)
	}
	return out, rows.Err()
}
