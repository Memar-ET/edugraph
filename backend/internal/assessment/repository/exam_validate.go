package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// ErrNotValidated means PublishExam was called before the exam ever went
// through ValidateExam -- status must be 'validation_pending', not just
// 'draft'.
var ErrNotValidated = errors.New("exam has not been validated yet")

type ExamForValidation struct {
	ID               uuid.UUID
	SchoolID         uuid.UUID
	SubjectCode      string
	GradeLevel       int
	ExamScope        string
	UnitNumbers      []int
	Status           string
	AttemptLimit     int
	TimeLimitMinutes *int
}

// FetchExamForValidation is the scope input for ValidateExam (subject/
// grade/scope/unit_numbers determine which curriculum.clos are "in scope",
// see FetchCurriculumCLOsInScope) and doubles as the general "fetch an exam
// by id" lookup for 2C's submit/answer-key paths -- SchoolID lets those
// verify a student/upload belongs to the same school as the exam.
func (r *Repository) FetchExamForValidation(ctx context.Context, examID uuid.UUID) (*ExamForValidation, error) {
	const q = `
		SELECT id, school_id, subject_code, grade_level, exam_scope, unit_numbers, status,
		       attempt_limit, time_limit_minutes
		FROM assessment.exams
		WHERE id = $1
	`
	var e ExamForValidation
	err := r.pool.QueryRow(ctx, q, examID).Scan(
		&e.ID, &e.SchoolID, &e.SubjectCode, &e.GradeLevel, &e.ExamScope, &e.UnitNumbers, &e.Status,
		&e.AttemptLimit, &e.TimeLimitMinutes,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("fetch exam for validation: %w", err)
	}
	return &e, nil
}

type QuestionWithClo struct {
	ID             uuid.UUID
	SequenceNumber int
	QuestionType   string
	Marks          int
	CloCode        *string
	BloomLevel     *string // from curriculum.clos, NULL if unmatched
	IsMandatory    *bool
	TopicID        *uuid.UUID // from topic_clo_mappings via clo_code
}

// FetchQuestionsWithClo joins each question to its linked CLO (if any) --
// Bloom level and mandatory-ness live on curriculum.clos, not on the
// question row itself (2A never copies them down), and topic linkage goes
// through topic_clo_mappings, not the always-NULL questions.topic_id
// column.
func (r *Repository) FetchQuestionsWithClo(ctx context.Context, examID uuid.UUID) ([]QuestionWithClo, error) {
	const q = `
		SELECT q.id, q.sequence_number, q.question_type, q.marks, q.clo_code,
		       c.bloom_level, c.is_mandatory, tcm.topic_id
		FROM assessment.questions q
		LEFT JOIN curriculum.clos c ON c.code = q.clo_code
		LEFT JOIN curriculum.topic_clo_mappings tcm ON tcm.clo_code = q.clo_code
		WHERE q.exam_id = $1
		ORDER BY q.sequence_number
	`
	rows, err := r.pool.Query(ctx, q, examID)
	if err != nil {
		return nil, fmt.Errorf("fetch questions with clo: %w", err)
	}
	defer rows.Close()

	var out []QuestionWithClo
	for rows.Next() {
		var qw QuestionWithClo
		if err := rows.Scan(&qw.ID, &qw.SequenceNumber, &qw.QuestionType, &qw.Marks, &qw.CloCode, &qw.BloomLevel, &qw.IsMandatory, &qw.TopicID); err != nil {
			return nil, fmt.Errorf("scan question with clo: %w", err)
		}
		out = append(out, qw)
	}
	return out, rows.Err()
}

type CLOInScope struct {
	Code        string
	BloomLevel  *string
	IsMandatory bool
	TopicID     uuid.UUID
	TopicTitle  string
	UnitNumber  int
}

// FetchCurriculumCLOsInScope returns the CLOs a Final Exam (unitNumbers
// empty -> whole subject/grade) or a Unit Test/Midterm (unitNumbers set ->
// just those units) is expected to cover. cardinality(...)=0 check treats
// both a NULL and an empty array the same way ("no scope narrowing").
func (r *Repository) FetchCurriculumCLOsInScope(ctx context.Context, subjectCode string, gradeLevel int, unitNumbers []int) ([]CLOInScope, error) {
	const q = `
		SELECT c.code, c.bloom_level, c.is_mandatory, t.id, t.title_en, u.number
		FROM curriculum.clos c
		JOIN curriculum.topic_clo_mappings tcm ON tcm.clo_code = c.code
		JOIN curriculum.topics t ON t.id = tcm.topic_id
		JOIN curriculum.units u ON u.id = t.unit_id
		WHERE c.subject_code = $1 AND c.grade_level = $2
		  AND (COALESCE(cardinality($3::int[]), 0) = 0 OR u.number = ANY($3::int[]))
		ORDER BY u.number, t.sequence_order, c.code
	`
	rows, err := r.pool.Query(ctx, q, subjectCode, gradeLevel, unitNumbers)
	if err != nil {
		return nil, fmt.Errorf("fetch curriculum clos in scope: %w", err)
	}
	defer rows.Close()

	var out []CLOInScope
	for rows.Next() {
		var c CLOInScope
		if err := rows.Scan(&c.Code, &c.BloomLevel, &c.IsMandatory, &c.TopicID, &c.TopicTitle, &c.UnitNumber); err != nil {
			return nil, fmt.Errorf("scan clo in scope: %w", err)
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

type TopicInScope struct {
	ID         uuid.UUID
	Title      string
	UnitNumber int
}

// FetchTopicsInScope is the topic-level counterpart of
// FetchCurriculumCLOsInScope, used for Topic Coverage -- includes topics
// with zero CLOs too (an inner join through topic_clo_mappings would hide
// them), so a topic can be flagged as untested even if it has no CLO at
// all yet.
func (r *Repository) FetchTopicsInScope(ctx context.Context, subjectCode string, gradeLevel int, unitNumbers []int) ([]TopicInScope, error) {
	const q = `
		SELECT t.id, t.title_en, u.number
		FROM curriculum.topics t
		JOIN curriculum.units u ON u.id = t.unit_id
		WHERE t.subject_code = $1 AND t.grade_level = $2
		  AND (COALESCE(cardinality($3::int[]), 0) = 0 OR u.number = ANY($3::int[]))
		ORDER BY u.number, t.sequence_order
	`
	rows, err := r.pool.Query(ctx, q, subjectCode, gradeLevel, unitNumbers)
	if err != nil {
		return nil, fmt.Errorf("fetch topics in scope: %w", err)
	}
	defer rows.Close()

	var out []TopicInScope
	for rows.Next() {
		var t TopicInScope
		if err := rows.Scan(&t.ID, &t.Title, &t.UnitNumber); err != nil {
			return nil, fmt.Errorf("scan topic in scope: %w", err)
		}
		out = append(out, t)
	}
	return out, rows.Err()
}

type PrerequisiteWarning struct {
	TopicID           uuid.UUID
	TopicTitle        string
	PrerequisiteID    uuid.UUID
	PrerequisiteTitle string
	PrerequisiteGrade int
	IsCrossGrade      bool
}

// FetchPrerequisiteWarnings looks up prerequisites of the topics a set of
// CLOs belong to. curriculum.topic_prerequisites is empty in this dataset
// today (confirmed live, no code path writes to it yet) so this correctly
// returns an empty slice -- the query itself is real and will start
// surfacing warnings the moment prerequisite data exists, no code change
// needed later.
func (r *Repository) FetchPrerequisiteWarnings(ctx context.Context, topicIDs []uuid.UUID) ([]PrerequisiteWarning, error) {
	if len(topicIDs) == 0 {
		return nil, nil
	}
	const q = `
		SELECT t.id, t.title_en, p.id, p.title_en, p.grade_level, tp.is_cross_grade
		FROM curriculum.topic_prerequisites tp
		JOIN curriculum.topics t ON t.id = tp.topic_id
		JOIN curriculum.topics p ON p.id = tp.prerequisite_id
		WHERE tp.topic_id = ANY($1)
	`
	rows, err := r.pool.Query(ctx, q, topicIDs)
	if err != nil {
		return nil, fmt.Errorf("fetch prerequisite warnings: %w", err)
	}
	defer rows.Close()

	var out []PrerequisiteWarning
	for rows.Next() {
		var w PrerequisiteWarning
		if err := rows.Scan(&w.TopicID, &w.TopicTitle, &w.PrerequisiteID, &w.PrerequisiteTitle, &w.PrerequisiteGrade, &w.IsCrossGrade); err != nil {
			return nil, fmt.Errorf("scan prerequisite warning: %w", err)
		}
		out = append(out, w)
	}
	return out, rows.Err()
}

// SaveValidationReport persists the computed report and moves the exam to
// 'validation_pending' -- the teacher reviews it, then calls PublishExam.
func (r *Repository) SaveValidationReport(ctx context.Context, examID uuid.UUID, reportJSON []byte) error {
	const q = `
		UPDATE assessment.exams
		SET validation_report = $2::jsonb, status = 'validation_pending', updated_at = now()
		WHERE id = $1
	`
	_, err := r.pool.Exec(ctx, q, examID, reportJSON)
	if err != nil {
		return fmt.Errorf("save validation report: %w", err)
	}
	return nil
}

// PublishExam only succeeds from 'validation_pending' -- a teacher must
// review a report (even an imperfect one; this isn't a hard quality gate)
// before publishing, matching the spec's described workflow.
func (r *Repository) PublishExam(ctx context.Context, examID uuid.UUID) error {
	const q = `
		UPDATE assessment.exams
		SET status = 'published', updated_at = now()
		WHERE id = $1 AND status = 'validation_pending'
	`
	tag, err := r.pool.Exec(ctx, q, examID)
	if err != nil {
		return fmt.Errorf("publish exam: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotValidated
	}
	return nil
}
