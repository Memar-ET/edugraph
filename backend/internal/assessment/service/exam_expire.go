package service

import (
	"context"

	"github.com/google/uuid"

	"github.com/edugraph-ai/edugraph/internal/assessment/repository"
)

// AutoSubmitExpiredAttempts finalizes every in_progress attempt whose
// server-set expires_at has passed -- called on a short ticker
// (internal/assessment/examworker) since there is no per-request trigger
// for "time just ran out" the way there is for a student action. Each
// attempt is handled independently; one failure must not block the rest
// from expiring, so errors are swallowed here and only the successful
// count is returned -- the caller logs failures via the returned error
// count if it wants detail, but a single bad attempt retries next tick
// rather than wedging the whole sweep.
func (s *Service) AutoSubmitExpiredAttempts(ctx context.Context) (succeeded, failed int) {
	expired, err := s.repo.FetchExpiredInProgressAttempts(ctx)
	if err != nil {
		return 0, 0
	}
	for _, a := range expired {
		if err := s.autoSubmitOne(ctx, a); err != nil {
			failed++
			continue
		}
		succeeded++
	}
	return succeeded, failed
}

// autoSubmitOne mirrors SubmitExam's freeze->grade->save->recompute-
// >notify sequence, but sources answers from the attempt's saved drafts
// (assessment.exam_draft_answers) instead of a request body -- there is
// no submitted answer set for an attempt that timed out, only whatever
// autosave last persisted. Unanswered questions (no draft row) are left
// out of the graded set entirely, which gradeMCQOrPend/RecomputeAttemptTotals
// already treat as 0 marks once the exam is finalized -- consistent with
// Part 14's "unanswered = zero" policy without needing separate logic.
func (s *Service) autoSubmitOne(ctx context.Context, a repository.ExpiredAttempt) error {
	ok, err := s.repo.MarkAttemptSubmitted(ctx, a.AttemptID, "time_expired", nil)
	if err != nil {
		return err
	}
	if !ok {
		// Already finalized by something else (e.g. the student's own
		// submit landed in the same instant) -- nothing to do.
		return nil
	}

	// Every question ASSIGNED to this attempt gets a graded row, not just
	// the ones with a draft -- an unanswered question must count as 0/
	// possible in the denominator (Part 14's "unanswered = zero" policy),
	// not be silently excluded from both totals the way iterating only
	// over drafts would. This mirrors what the frontend already does for
	// a manual submit (one AnswerInput per question, "" when unanswered).
	assigned, err := s.repo.FetchAttemptQuestions(ctx, a.AttemptID)
	if err != nil {
		return err
	}
	drafts, err := s.repo.FetchDraftAnswers(ctx, a.AttemptID)
	if err != nil {
		return err
	}
	draftByQuestion := make(map[string]string, len(drafts))
	for _, d := range drafts {
		draftByQuestion[d.QuestionID.String()] = d.Response
	}

	questions, err := s.repo.FetchQuestionsForGrading(ctx, a.ExamID)
	if err != nil {
		return err
	}
	questionsByID := make(map[uuid.UUID]repository.QuestionForGrading, len(questions))
	for _, q := range questions {
		questionsByID[q.ID] = q
	}

	answers := make([]repository.GradedAnswer, 0, len(assigned))
	for _, sq := range assigned {
		id, err := uuid.Parse(sq.ID)
		if err != nil {
			continue
		}
		q, ok := questionsByID[id]
		if !ok {
			continue
		}
		answers = append(answers, gradeMCQOrPend(q, draftByQuestion[sq.ID]))
	}

	if err := s.repo.SaveStudentAnswers(ctx, a.AttemptID, a.StudentID, a.SchoolID, answers, nil); err != nil {
		return err
	}
	if err := s.repo.RecomputeAttemptTotals(ctx, a.AttemptID); err != nil {
		return err
	}
	if total, _, _, err := s.repo.FetchAttemptTotals(ctx, a.AttemptID); err == nil && total != nil {
		s.enqueueGapAnalysis(ctx, a.AttemptID)
		s.recordLearningEventsAndTrace(ctx, a.AttemptID)
	}
	return nil
}
