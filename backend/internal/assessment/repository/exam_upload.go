package repository

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/edugraph-ai/edugraph/internal/assessment/dto"
)

// ErrSubjectNotMatched means no curriculum.subjects row for the derived
// grade level had a name/code appearing in the exam title -- either the
// curriculum for that subject/grade hasn't been uploaded yet, or the title
// doesn't clearly name a known subject.
var ErrSubjectNotMatched = errors.New("no matching curriculum subject")

// TeacherSchoolID resolves the uploader's school, needed for the NOT NULL
// assessment.exams.school_id column. users.school_id (V003) covers both
// teacher and school_admin uploaders.
func (r *Repository) TeacherSchoolID(ctx context.Context, userID uuid.UUID) (uuid.UUID, error) {
	var schoolID *uuid.UUID
	err := r.pool.QueryRow(ctx, `SELECT school_id FROM users WHERE id = $1`, userID).Scan(&schoolID)
	if errors.Is(err, pgx.ErrNoRows) {
		return uuid.Nil, ErrNotFound
	}
	if err != nil {
		return uuid.Nil, fmt.Errorf("lookup uploader school: %w", err)
	}
	if schoolID == nil {
		return uuid.Nil, fmt.Errorf("uploader has no school_id set")
	}
	return *schoolID, nil
}

// MatchSubjectFromTitle finds the curriculum.subjects row for gradeLevel
// whose name_en or code appears in the exam title (case-insensitive), e.g.
// title "Grade 11 Biology Unit Test" + gradeLevel 11 matches the subject
// row {code: "BIO", name_en: "Biology", grade_level: 11}. This both derives
// the subject and validates that a curriculum actually exists for it --
// the whole point of deriving subject/grade from the title instead of a
// free-text field the teacher fills in.
func (r *Repository) MatchSubjectFromTitle(ctx context.Context, title string, gradeLevel int) (string, error) {
	rows, err := r.pool.Query(ctx, `SELECT code, name_en FROM curriculum.subjects WHERE grade_level = $1`, gradeLevel)
	if err != nil {
		return "", fmt.Errorf("query curriculum subjects: %w", err)
	}
	defer rows.Close()

	lowerTitle := strings.ToLower(title)
	for rows.Next() {
		var code, nameEn string
		if err := rows.Scan(&code, &nameEn); err != nil {
			return "", fmt.Errorf("scan subject: %w", err)
		}
		if strings.Contains(lowerTitle, strings.ToLower(nameEn)) || strings.Contains(lowerTitle, strings.ToLower(code)) {
			return code, nil
		}
	}
	if err := rows.Err(); err != nil {
		return "", fmt.Errorf("iterate subjects: %w", err)
	}
	return "", ErrSubjectNotMatched
}

// CreateExam inserts a new exam row with status 'pending', mirroring
// curriculum.CreateJob -- the AI worker flips it to 'parsing' then
// 'draft'/'failed' once it picks the job off queue:exam:parse.
func (r *Repository) CreateExam(
	ctx context.Context,
	teacherID, schoolID uuid.UUID,
	req dto.UploadExamRequest,
	examScope, subjectCode string,
	gradeLevel int,
	unitNumbers []int,
	fileRef, fileName string,
) (uuid.UUID, error) {
	var examID uuid.UUID
	const q = `
		INSERT INTO assessment.exams
			(created_by, school_id, subject_code, grade_level, academic_year, term, title, total_marks, exam_scope, unit_numbers, file_s3_key, status)
		VALUES
			($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, 'pending')
		RETURNING id
	`
	err := r.pool.QueryRow(ctx, q,
		teacherID, schoolID, subjectCode, gradeLevel, req.AcademicYear, req.Term, req.Title, req.TotalMarks, examScope, unitNumbers, fileRef,
	).Scan(&examID)
	if err != nil {
		return uuid.Nil, fmt.Errorf("create exam: %w", err)
	}
	// fileName is currently unused beyond upload -- assessment.exams has no
	// file_name column (unlike curriculum.upload_jobs); kept as a parameter
	// for parity with CreateJob's signature and in case it's needed later.
	_ = fileName
	return examID, nil
}

// GetExam backs GET /api/v1/exams/:id -- status + question count so the
// caller can poll while parsing runs.
func (r *Repository) GetExam(ctx context.Context, examID uuid.UUID) (*dto.ExamStatus, error) {
	const q = `
		SELECT e.id, e.status, e.title, e.subject_code, e.grade_level, e.exam_scope,
		       e.academic_year, e.total_marks, e.parse_error, e.created_at, e.validation_report,
		       (SELECT count(*) FROM assessment.questions q WHERE q.exam_id = e.id)
		FROM assessment.exams e
		WHERE e.id = $1
	`
	var s dto.ExamStatus
	var reportJSON []byte
	err := r.pool.QueryRow(ctx, q, examID).Scan(
		&s.ExamID, &s.Status, &s.Title, &s.SubjectCode, &s.GradeLevel, &s.ExamScope,
		&s.AcademicYear, &s.TotalMarks, &s.ParseError, &s.CreatedAt, &reportJSON, &s.QuestionCount,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get exam: %w", err)
	}
	if reportJSON != nil {
		var report dto.ValidationReport
		if err := json.Unmarshal(reportJSON, &report); err != nil {
			return nil, fmt.Errorf("unmarshal validation report: %w", err)
		}
		s.ValidationReport = &report
	}
	return &s, nil
}
