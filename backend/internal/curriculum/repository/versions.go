package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"

	"github.com/edugraph-ai/edugraph/internal/curriculum/dto"
	apperrors "github.com/edugraph-ai/edugraph/pkg/errors"
)

type subjectVersionCore struct {
	code                string
	version             int
	isCurrent           bool
	previousVersionCode *string
}

func (r *Repository) fetchSubjectVersionCore(ctx context.Context, tx pgx.Tx, code string) (*subjectVersionCore, error) {
	var v subjectVersionCore
	err := tx.QueryRow(ctx,
		`SELECT code, version, is_current, previous_version_code FROM curriculum.subjects WHERE code = $1 FOR UPDATE`,
		code,
	).Scan(&v.code, &v.version, &v.isCurrent, &v.previousVersionCode)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, apperrors.NotFound(fmt.Sprintf("subject %q not found", code))
	}
	if err != nil {
		return nil, fmt.Errorf("fetch subject %q: %w", code, err)
	}
	return &v, nil
}

// MarkAsRevision links newCode as the version that supersedes previousCode
// (feature 1.3): previousCode is flipped to IsCurrent=false/SupersededAt=now,
// and newCode becomes IsCurrent=true with version = previousCode's version + 1
// and PreviousVersionCode = previousCode. Both subjects must already exist
// (i.e. newCode was uploaded and approved through the normal pipeline under
// its own code) -- this call only records the lineage, it never copies or
// mutates units/topics/clos.
func (r *Repository) MarkAsRevision(ctx context.Context, newCode, previousCode string) (*dto.SubjectVersion, error) {
	if newCode == previousCode {
		return nil, apperrors.BadRequest("a subject cannot supersede itself")
	}

	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	previous, err := r.fetchSubjectVersionCore(ctx, tx, previousCode)
	if err != nil {
		return nil, err
	}
	if !previous.isCurrent {
		return nil, apperrors.Conflict(fmt.Sprintf(
			"%q is not the current version of its lineage; supersede its current version instead", previousCode,
		))
	}

	next, err := r.fetchSubjectVersionCore(ctx, tx, newCode)
	if err != nil {
		return nil, err
	}
	if next.previousVersionCode != nil {
		return nil, apperrors.Conflict(fmt.Sprintf("%q is already linked as a revision of %q", newCode, *next.previousVersionCode))
	}

	if _, err := tx.Exec(ctx,
		`UPDATE curriculum.subjects SET is_current = false, superseded_at = now() WHERE code = $1`,
		previousCode,
	); err != nil {
		return nil, fmt.Errorf("supersede subject %q: %w", previousCode, err)
	}

	newVersion := previous.version + 1
	var resp dto.SubjectVersion
	err = tx.QueryRow(ctx, `
		UPDATE curriculum.subjects
		SET version = $2, previous_version_code = $3, is_current = true
		WHERE code = $1
		RETURNING code, version, is_current, academic_year, previous_version_code, superseded_at, created_at
	`, newCode, newVersion, previousCode).Scan(
		&resp.Code, &resp.Version, &resp.IsCurrent, &resp.AcademicYear,
		&resp.PreviousVersionCode, &resp.SupersededAt, &resp.CreatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("promote subject %q to version %d: %w", newCode, newVersion, err)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit tx: %w", err)
	}
	return &resp, nil
}

// GetVersionHistory walks a subject's lineage backward from `code` via
// previous_version_code and returns it oldest-first. `code` itself is
// included (it need not be the current version).
func (r *Repository) GetVersionHistory(ctx context.Context, code string) ([]dto.SubjectVersion, error) {
	const maxDepth = 100 // guards against a corrupted/cyclic chain

	history := make([]dto.SubjectVersion, 0, 4)
	cursor := &code
	for i := 0; i < maxDepth && cursor != nil; i++ {
		var v dto.SubjectVersion
		err := r.pool.QueryRow(ctx, `
			SELECT code, version, is_current, academic_year, previous_version_code, superseded_at, created_at
			FROM curriculum.subjects WHERE code = $1
		`, *cursor).Scan(
			&v.Code, &v.Version, &v.IsCurrent, &v.AcademicYear,
			&v.PreviousVersionCode, &v.SupersededAt, &v.CreatedAt,
		)
		if errors.Is(err, pgx.ErrNoRows) {
			if i == 0 {
				return nil, apperrors.NotFound(fmt.Sprintf("subject %q not found", code))
			}
			break
		}
		if err != nil {
			return nil, fmt.Errorf("fetch subject %q: %w", *cursor, err)
		}
		history = append(history, v)
		cursor = v.PreviousVersionCode
	}

	// Reverse in place so the response reads oldest -> newest.
	for i, j := 0, len(history)-1; i < j; i, j = i+1, j-1 {
		history[i], history[j] = history[j], history[i]
	}
	return history, nil
}
