package service

import (
	"context"
	"errors"

	"github.com/google/uuid"

	"github.com/edugraph-ai/edugraph/internal/assessment/dto"
	"github.com/edugraph-ai/edugraph/internal/assessment/repository"
	"github.com/edugraph-ai/edugraph/pkg/ai"
	apperrors "github.com/edugraph-ai/edugraph/pkg/errors"
)

// AskTutor proxies a student's question to the ai-service Graph-RAG
// tutor (Capability 3C), which injects the student's own gap records and
// prerequisite context into the Gemini prompt. Synchronous by design --
// it's a chat interaction, not a pipeline job.
func (s *Service) AskTutor(ctx context.Context, userID uuid.UUID, req dto.TutorAskRequest) (*ai.TutorAskResponse, error) {
	student, err := s.repo.FetchStudentProfile(ctx, userID)
	if errors.Is(err, repository.ErrNotFound) {
		return nil, apperrors.Forbidden("no student profile for this account")
	}
	if err != nil {
		return nil, apperrors.Internal(err)
	}

	language := "en"
	if req.Language != nil {
		language = *req.Language
	}

	resp, err := s.ai.TutorAsk(ctx, ai.TutorAskRequest{
		StudentID: student.ID.String(),
		Question:  req.Question,
		Language:  language,
	})
	if err != nil {
		// 503 from the ai-service (no Gemini key / model down) and network
		// failures both land here -- the tutor is genuinely unavailable.
		return nil, apperrors.Internal(err)
	}
	return resp, nil
}
