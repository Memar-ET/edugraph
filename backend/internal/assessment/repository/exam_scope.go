package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/edugraph-ai/edugraph/internal/assessment/dto"
)

// ErrUnknownSubjectCode means the caller supplied a subjectCode that
// doesn't exist in curriculum.subjects for the (possibly also-updated)
// grade level -- the same guard UploadExam's MatchSubjectFromTitle
// effectively applies at upload time, just checking an explicit code
// instead of matching one out of a title string.
var ErrUnknownSubjectCode = errors.New("subject code does not exist for that grade level")

// UpdateExamScope applies a partial correction to an exam's
// subject/grade/scope/unit_numbers (see dto.UpdateExamScopeRequest) inside
// a single row-locked transaction, and reports whether subjectCode or
// gradeLevel actually changed value -- the caller (service.UpdateExamScope)
// uses that to decide whether a CLO rematch needs to be queued. Changing
// examScope/unitNumbers alone only affects which CLOs count as "in scope"
// for 2B's report (recomputed live on every POST .../validate call); it
// doesn't invalidate any question's already-assigned CLO match.
func (r *Repository) UpdateExamScope(
	ctx context.Context, examID uuid.UUID, req dto.UpdateExamScopeRequest,
) (updated ExamForValidation, subjectOrGradeChanged bool, err error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return ExamForValidation{}, false, fmt.Errorf("begin update exam scope tx: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var current ExamForValidation
	const selectQ = `
		SELECT id, school_id, subject_code, grade_level, exam_scope, unit_numbers, status
		FROM assessment.exams WHERE id = $1 FOR UPDATE
	`
	err = tx.QueryRow(ctx, selectQ, examID).Scan(
		&current.ID, &current.SchoolID, &current.SubjectCode, &current.GradeLevel,
		&current.ExamScope, &current.UnitNumbers, &current.Status,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return ExamForValidation{}, false, ErrNotFound
	}
	if err != nil {
		return ExamForValidation{}, false, fmt.Errorf("fetch exam for scope update: %w", err)
	}

	next := current
	if req.GradeLevel != nil {
		next.GradeLevel = *req.GradeLevel
	}
	if req.SubjectCode != nil {
		next.SubjectCode = *req.SubjectCode
	}
	if req.ExamScope != nil {
		next.ExamScope = *req.ExamScope
	}
	if req.UnitNumbers != nil {
		next.UnitNumbers = req.UnitNumbers
	}

	if next.SubjectCode != current.SubjectCode || next.GradeLevel != current.GradeLevel {
		var exists bool
		const checkQ = `SELECT EXISTS(SELECT 1 FROM curriculum.subjects WHERE code = $1 AND grade_level = $2)`
		if err := tx.QueryRow(ctx, checkQ, next.SubjectCode, next.GradeLevel).Scan(&exists); err != nil {
			return ExamForValidation{}, false, fmt.Errorf("check subject exists: %w", err)
		}
		if !exists {
			return ExamForValidation{}, false, ErrUnknownSubjectCode
		}
	}

	const updateQ = `
		UPDATE assessment.exams
		SET subject_code = $2, grade_level = $3, exam_scope = $4, unit_numbers = $5, updated_at = now()
		WHERE id = $1
	`
	if _, err := tx.Exec(ctx, updateQ, examID, next.SubjectCode, next.GradeLevel, next.ExamScope, next.UnitNumbers); err != nil {
		return ExamForValidation{}, false, fmt.Errorf("update exam scope: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return ExamForValidation{}, false, fmt.Errorf("commit exam scope update: %w", err)
	}

	changed := next.SubjectCode != current.SubjectCode || next.GradeLevel != current.GradeLevel
	return next, changed, nil
}
