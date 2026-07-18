package service

import (
	"context"
	"errors"
	"fmt"
	"io"

	"github.com/edugraph-ai/edugraph/internal/assessment/dto"
	"github.com/edugraph-ai/edugraph/internal/assessment/repository"
	apperrors "github.com/edugraph-ai/edugraph/pkg/errors"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// UploadExam mirrors curriculum's Upload workflow: derive grade/scope/
// subject from the title, validate the declared total against the scope,
// save the file to storage, create the exam row, then queue it for AI
// parsing via Redis.
func (s *Service) UploadExam(
	ctx context.Context,
	userID uuid.UUID,
	req dto.UploadExamRequest,
	fileName, mimeType string,
	fileSize int64,
	file io.Reader,
) (*dto.UploadExamResponse, error) {
	gradeLevel, examScope, ok := deriveGradeAndScope(req.Title)
	if !ok {
		return nil, apperrors.BadRequest(
			"could not determine grade level and exam type from the title -- include both, " +
				`e.g. "Grade 11 Biology Unit Test - Cell Biology" (unit test / midterm / final exam)`,
		)
	}

	if !totalMarksAllowed(examScope, req.TotalMarks) {
		return nil, apperrors.BadRequest(fmt.Sprintf(
			"totalMarks %d is not valid for a %s (allowed: %v)", req.TotalMarks, examScope, allowedTotalMarks[examScope],
		))
	}

	subjectCode, err := s.repo.MatchSubjectFromTitle(ctx, req.Title, gradeLevel)
	if errors.Is(err, repository.ErrSubjectNotMatched) {
		return nil, apperrors.BadRequest(fmt.Sprintf(
			"could not match a curriculum subject for grade %d in the exam title -- make sure a curriculum "+
				"has been uploaded for that grade/subject and the title includes the subject name", gradeLevel,
		))
	}
	if err != nil {
		return nil, apperrors.Internal(err)
	}

	schoolID, err := s.repo.TeacherSchoolID(ctx, userID)
	if errors.Is(err, repository.ErrNotFound) {
		return nil, apperrors.BadRequest("uploader has no school on record")
	}
	if err != nil {
		return nil, apperrors.Internal(err)
	}

	fileRef, err := s.storage.Upload(ctx, fileName, mimeType, file)
	if err != nil {
		return nil, fmt.Errorf("storage upload failed: %w", err)
	}

	unitNumbers := deriveUnitNumbers(req.Title)

	examID, err := s.repo.CreateExam(ctx, userID, schoolID, req, examScope, subjectCode, gradeLevel, unitNumbers, fileRef, fileName)
	if err != nil {
		return nil, fmt.Errorf("create exam record failed: %w", err)
	}

	// Non-fatal: mirrors curriculum's queue push -- the exam row exists at
	// status 'pending' even if this fails.
	if err := s.redis.LPush(ctx, "queue:exam:parse", examID.String()).Err(); err != nil {
		fmt.Printf("⚠️ Redis queue push failed for exam %s: %v\n", examID, err)
	}

	return &dto.UploadExamResponse{
		ExamID:  examID,
		Status:  "pending",
		Message: "File uploaded successfully. Parsing queued.",
	}, nil
}

func (s *Service) GetExam(ctx context.Context, examID uuid.UUID) (*dto.ExamStatus, error) {
	exam, err := s.repo.GetExam(ctx, examID)
	if errors.Is(err, repository.ErrNotFound) || errors.Is(err, pgx.ErrNoRows) {
		return nil, apperrors.NotFound("exam not found")
	}
	if err != nil {
		return nil, apperrors.Internal(err)
	}
	return exam, nil
}
