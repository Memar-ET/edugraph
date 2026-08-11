package service

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"

	"github.com/edugraph-ai/edugraph/internal/assessment/dto"
	"github.com/edugraph-ai/edugraph/internal/assessment/repository"
	apperrors "github.com/edugraph-ai/edugraph/pkg/errors"
)

// UpdateExamScope lets a teacher correct a wrong subject/grade/exam-type/
// unit-range without re-uploading the document -- see title_parser.go's
// deriveGradeAndScope comment: the title is only ever a fallback guess for
// what the exam covers, and until now there was no way to fix a wrong
// guess except re-uploading the whole file with a better title. Only
// allowed before the exam is published: an exam already delivered to
// students shouldn't have its curriculum scope moved out from under
// attempts that are submitted or in progress.
func (s *Service) UpdateExamScope(ctx context.Context, examID uuid.UUID, req dto.UpdateExamScopeRequest) (*dto.UpdateExamScopeResponse, error) {
	if req.SubjectCode == nil && req.GradeLevel == nil && req.ExamScope == nil && req.UnitNumbers == nil {
		return nil, apperrors.BadRequest("at least one of subjectCode, gradeLevel, examScope, unitNumbers must be supplied")
	}

	exam, err := s.repo.FetchExamForValidation(ctx, examID)
	if errors.Is(err, repository.ErrNotFound) {
		return nil, apperrors.NotFound("exam not found")
	}
	if err != nil {
		return nil, apperrors.Internal(err)
	}
	if exam.Status == "published" || exam.Status == "closed" {
		return nil, apperrors.Conflict(fmt.Sprintf("cannot change scope on a %s exam", exam.Status))
	}

	updated, subjectOrGradeChanged, err := s.repo.UpdateExamScope(ctx, examID, req)
	if errors.Is(err, repository.ErrNotFound) {
		return nil, apperrors.NotFound("exam not found")
	}
	if errors.Is(err, repository.ErrUnknownSubjectCode) {
		return nil, apperrors.BadRequest("no curriculum subject with that code exists for the given grade level")
	}
	if err != nil {
		return nil, apperrors.Internal(err)
	}

	resp := &dto.UpdateExamScopeResponse{
		ExamID:      updated.ID,
		SubjectCode: updated.SubjectCode,
		GradeLevel:  updated.GradeLevel,
		ExamScope:   updated.ExamScope,
		UnitNumbers: updated.UnitNumbers,
		Message:     "Exam scope updated. Re-run POST .../validate to refresh the compliance report against the corrected scope.",
	}

	// Subject or grade changing invalidates every question's existing CLO
	// match -- 2A matched each one against the *old* subject's CLO set.
	// Queue a rematch against the corrected subject instead of making the
	// teacher re-upload the file just to get correct CLO assignments.
	if subjectOrGradeChanged {
		s.enqueueCLORematch(ctx, examID)
		resp.CloRematchQueued = true
		resp.Message += " Subject/grade changed, so question-to-CLO matches are being recomputed in the background -- wait a few seconds before validating."
	}

	return resp, nil
}

// enqueueCLORematch pushes the exam id onto queue:exam:rematch for
// ai-service/app/workers/exam_rematch_worker.py, which re-runs CLO
// matching against the already-extracted question text (no re-parsing of
// the original file, unlike queue:exam:parse). Non-fatal like every other
// queue push in this package: the scope change already committed, and
// matching can be retried later if Redis is briefly down.
func (s *Service) enqueueCLORematch(ctx context.Context, examID uuid.UUID) {
	if err := s.redis.LPush(ctx, "queue:exam:rematch", examID.String()).Err(); err != nil {
		fmt.Printf("⚠️ Redis queue push failed for CLO rematch of exam %s: %v\n", examID, err)
	}
}
