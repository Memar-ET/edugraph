package service

import (
	"context"
	"errors"
	"fmt"
	"io"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/edugraph-ai/edugraph/internal/assessment/dto"
	"github.com/edugraph-ai/edugraph/internal/assessment/repository"
	apperrors "github.com/edugraph-ai/edugraph/pkg/errors"
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

func (s *Service) GetExam(ctx context.Context, userID, examID uuid.UUID) (*dto.ExamStatus, error) {
	if err := s.verifyCallerOwnsExam(ctx, userID, examID); err != nil {
		return nil, err
	}
	exam, err := s.repo.GetExam(ctx, examID)
	if errors.Is(err, repository.ErrNotFound) || errors.Is(err, pgx.ErrNoRows) {
		return nil, apperrors.NotFound("exam not found")
	}
	if err != nil {
		return nil, apperrors.Internal(err)
	}
	return exam, nil
}

// verifyCallerOwnsExam is the teacher/school_admin-side counterpart to
// verifyStudentAccess (exam_submit.go) -- checklist 11.3. Every exam-
// management function below used to check role only (middleware.
// RequireRole(teacher, school_admin)), with no check that the caller's
// OWN school is the exam's school; any teacher/school_admin nationwide
// could read, validate, publish, grade, or print any exam by guessing/
// enumerating its UUID. This was a documented, deliberate convention at
// the time (see the old comment on ListQuestionsForGrading), not an
// oversight -- fixed here, applied uniformly rather than function by
// function, since the gap was identical everywhere.
//
// Deliberately checks against repo.ExamSchoolID (a narrow, standalone
// lookup) rather than each function's own richer exam-fetch query, so
// this drops in as a single line at the top of each function without
// changing any of their existing queries/DTOs.
func (s *Service) verifyCallerOwnsExam(ctx context.Context, userID, examID uuid.UUID) error {
	examSchoolID, err := s.repo.ExamSchoolID(ctx, examID)
	if errors.Is(err, repository.ErrNotFound) {
		return apperrors.NotFound("exam not found")
	}
	if err != nil {
		return apperrors.Internal(err)
	}
	callerSchoolID, err := s.repo.TeacherSchoolID(ctx, userID)
	if errors.Is(err, repository.ErrNotFound) {
		return apperrors.Forbidden("no school on record for this account")
	}
	if err != nil {
		return apperrors.Internal(err)
	}
	if callerSchoolID != examSchoolID {
		return apperrors.Forbidden("exam belongs to a different school")
	}
	return nil
}
