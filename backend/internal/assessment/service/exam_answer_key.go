package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"

	"github.com/google/uuid"

	"github.com/edugraph-ai/edugraph/internal/assessment/dto"
	"github.com/edugraph-ai/edugraph/internal/assessment/repository"
	apperrors "github.com/edugraph-ai/edugraph/pkg/errors"
)

// answerKeyJobPayload is pushed as JSON, not a bare id string like the
// other two queues -- applying an answer key needs both which exam it's
// for and where the uploaded file's bytes live. See
// ai-service/app/workers/answer_key_worker.py, the only place that parses
// this shape.
type answerKeyJobPayload struct {
	ExamID  string `json:"examId"`
	FileRef string `json:"fileRef"`
}

// UploadAnswerKey handles a separately-uploaded answer-key document for an
// exam that's already been parsed (2A) -- real exam docs turned out to
// ship the Q#/Correct-Answer table as its own document, not embedded in
// the student-facing exam paper. Queues async processing the same way
// UploadExam does, since the table extraction needs ai-service's
// PDF/DOCX libraries.
func (s *Service) UploadAnswerKey(
	ctx context.Context,
	userID, examID uuid.UUID,
	fileName, mimeType string,
	file io.Reader,
) (*dto.UploadAnswerKeyResponse, error) {
	if err := s.verifyCallerOwnsExam(ctx, userID, examID); err != nil {
		return nil, err
	}
	if _, err := s.repo.FetchExamForValidation(ctx, examID); err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, apperrors.NotFound("exam not found")
		}
		return nil, apperrors.Internal(err)
	}

	fileRef, err := s.storage.Upload(ctx, fileName, mimeType, file)
	if err != nil {
		return nil, fmt.Errorf("storage upload failed: %w", err)
	}

	payload, err := json.Marshal(answerKeyJobPayload{ExamID: examID.String(), FileRef: fileRef})
	if err != nil {
		return nil, apperrors.Internal(err)
	}

	// Non-fatal: mirrors the other queue pushes -- the file is stored even
	// if this fails, just not processed until retried.
	if err := s.redis.LPush(ctx, "queue:exam:answerkey", payload).Err(); err != nil {
		fmt.Printf("⚠️ Redis queue push failed for answer key on exam %s: %v\n", examID, err)
	}

	return &dto.UploadAnswerKeyResponse{
		ExamID:  examID,
		Message: "Answer key uploaded successfully. Applying to matching questions.",
	}, nil
}
