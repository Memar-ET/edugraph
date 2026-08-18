package service

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/edugraph-ai/edugraph/internal/assessment/dto"
	"github.com/edugraph-ai/edugraph/internal/assessment/repository"
	apperrors "github.com/edugraph-ai/edugraph/pkg/errors"
)

// verifyStudentAccess is shared by everything a student does with an exam
// (viewing its questions, submitting answers): the exam must be published,
// and the student's school/grade must match it. Returns apperrors already
// mapped, ready for a handler/service caller to return directly.
func (s *Service) verifyStudentAccess(ctx context.Context, userID, examID uuid.UUID) (*repository.ExamForValidation, *repository.StudentProfile, error) {
	exam, err := s.repo.FetchExamForValidation(ctx, examID)
	if errors.Is(err, repository.ErrNotFound) {
		return nil, nil, apperrors.NotFound("exam not found")
	}
	if err != nil {
		return nil, nil, apperrors.Internal(err)
	}
	if exam.Status != "published" {
		return nil, nil, apperrors.Conflict("exam is not published yet")
	}

	student, err := s.repo.FetchStudentProfile(ctx, userID)
	if errors.Is(err, repository.ErrNotFound) {
		return nil, nil, apperrors.Forbidden("no student profile for this account")
	}
	if err != nil {
		return nil, nil, apperrors.Internal(err)
	}
	if student.SchoolID != exam.SchoolID || student.GradeLevel != exam.GradeLevel {
		return nil, nil, apperrors.Forbidden("this exam is not available to your class")
	}

	return exam, student, nil
}

// ListExamQuestionsForStudent is what the exam-taking page fetches while
// answering -- dto.StudentQuestion has no answer_key/clo_code fields,
// unlike the grading-side QuestionForGrading. Requires an in_progress
// attempt (created by StartAttempt) and returns that attempt's persisted
// question/option order, never a fresh randomization -- a refresh must
// always see the exact same form.
func (s *Service) ListExamQuestionsForStudent(ctx context.Context, userID, examID uuid.UUID) ([]dto.StudentQuestion, error) {
	_, student, err := s.verifyStudentAccess(ctx, userID, examID)
	if err != nil {
		return nil, err
	}
	attempt, err := s.repo.FindInProgressAttempt(ctx, student.ID, examID)
	if err != nil {
		return nil, apperrors.Internal(err)
	}
	if attempt == nil {
		return nil, apperrors.Conflict("start the exam (POST .../start) before viewing its questions")
	}
	questions, err := s.repo.FetchAttemptQuestions(ctx, attempt.AttemptID)
	if err != nil {
		return nil, apperrors.Internal(err)
	}
	return questions, nil
}

// ListQuestionsForGrading backs the teacher-facing "Grade Exam" spreadsheet's
// column headers. Used to have no per-teacher-school ownership check --
// see verifyCallerOwnsExam's doc comment (exam_upload.go) for why that
// was fixed (checklist 11.3).
func (s *Service) ListQuestionsForGrading(ctx context.Context, userID, examID uuid.UUID) ([]dto.GradingQuestion, error) {
	if err := s.verifyCallerOwnsExam(ctx, userID, examID); err != nil {
		return nil, err
	}
	questions, err := s.repo.FetchQuestionsForGrading(ctx, examID)
	if err != nil {
		return nil, apperrors.Internal(err)
	}
	out := make([]dto.GradingQuestion, 0, len(questions))
	for _, q := range questions {
		out = append(out, dto.GradingQuestion{
			ID:             q.ID.String(),
			SequenceNumber: q.SequenceNumber,
			QuestionText:   q.QuestionText,
			QuestionType:   q.QuestionType,
			Marks:          q.Marks,
			PartLabel:      q.PartLabel,
			AnswerKey:      q.AnswerKey,
			Options:        q.Options,
		})
	}
	return out, nil
}

// SubmitExam is Flow 1 (digital): the student answers directly in the
// frontend. MCQ questions with a parsed answer_key are graded instantly;
// everything else (essay/short_answer/long_answer/calculation, or an MCQ
// with no answer key) is left pending -- inferred from marks_awarded IS
// NULL, there's no status column for this.
//
// Requires an in_progress attempt (created by StartAttempt) -- attempts
// are no longer created here. Order matches Part 13's transactional
// contract: verify ownership/state/server-time (readonly, safe to do
// before the atomic freeze), THEN freeze via MarkAttemptSubmitted (the
// actual concurrency guard -- WHERE status='in_progress' means exactly
// one concurrent submit can win), THEN persist answers/grade. A losing
// concurrent request, or a network-retried request carrying the same
// idempotencyKey, gets the SAME result back rather than an error or a
// second grading pass.
func (s *Service) SubmitExam(ctx context.Context, userID, examID uuid.UUID, req dto.SubmitExamRequest) (*dto.SubmitExamResponse, error) {
	_, student, err := s.verifyStudentAccess(ctx, userID, examID)
	if err != nil {
		return nil, err
	}

	attempt, err := s.repo.FindInProgressAttempt(ctx, student.ID, examID)
	if err != nil {
		return nil, apperrors.Internal(err)
	}
	if attempt == nil {
		if req.IdempotencyKey != nil {
			if resp, found, err := s.repo.FetchSubmitResultByIdempotencyKey(ctx, student.ID, examID, *req.IdempotencyKey); err != nil {
				return nil, apperrors.Internal(err)
			} else if found {
				return resp, nil
			}
		}
		return nil, apperrors.Conflict("no exam session in progress -- start the exam first, or it has already been submitted")
	}

	// A request landing at/after expires_at is graded and finalized the
	// same as any other submit, just tagged time_expired instead of
	// student_submit -- rejecting it here would only force the student
	// to wait out the auto-submit ticker's own poll interval for a
	// request that already carries their real, intended answers. The
	// atomic MarkAttemptSubmitted guard below is what actually prevents
	// this from racing the ticker (or a concurrent duplicate submit),
	// not this check.
	submissionReason := "student_submit"
	if attempt.ExpiresAt != nil && !time.Now().UTC().Before(*attempt.ExpiresAt) {
		submissionReason = "time_expired"
	}

	assigned, err := s.repo.FetchAttemptQuestions(ctx, attempt.AttemptID)
	if err != nil {
		return nil, apperrors.Internal(err)
	}
	assignedIDs := make(map[uuid.UUID]bool, len(assigned))
	for _, q := range assigned {
		if id, err := uuid.Parse(q.ID); err == nil {
			assignedIDs[id] = true
		}
	}
	for _, a := range req.Answers {
		if !assignedIDs[a.QuestionID] {
			return nil, apperrors.BadRequest(fmt.Sprintf("question %s is not assigned to this attempt", a.QuestionID))
		}
	}

	questions, err := s.repo.FetchQuestionsForGrading(ctx, examID)
	if err != nil {
		return nil, apperrors.Internal(err)
	}
	questionsByID := make(map[uuid.UUID]repository.QuestionForGrading, len(questions))
	for _, q := range questions {
		questionsByID[q.ID] = q
	}

	ok, err := s.repo.MarkAttemptSubmitted(ctx, attempt.AttemptID, submissionReason, req.IdempotencyKey)
	if err != nil {
		return nil, apperrors.Internal(err)
	}
	if !ok {
		// Lost the freeze race to a concurrent submit for this same
		// attempt -- return its result rather than erroring or
		// re-grading.
		return s.repo.FetchSubmitSummary(ctx, attempt.AttemptID)
	}

	answers := make([]repository.GradedAnswer, 0, len(req.Answers))
	graded, pending := 0, 0
	for _, a := range req.Answers {
		q := questionsByID[a.QuestionID]
		ga := gradeMCQOrPend(q, a.Response)
		ga.TimeSpentSecs = a.TimeSpentSecs
		answers = append(answers, ga)
		if ga.MarksAwarded != nil {
			graded++
		} else {
			pending++
		}
	}

	if err := s.repo.SaveStudentAnswers(ctx, attempt.AttemptID, student.ID, student.SchoolID, answers, nil); err != nil {
		return nil, apperrors.Internal(err)
	}
	if err := s.repo.RecomputeAttemptTotals(ctx, attempt.AttemptID); err != nil {
		return nil, apperrors.Internal(err)
	}

	resp := &dto.SubmitExamResponse{
		AttemptID:           attempt.AttemptID,
		GradedCount:         graded,
		PendingGradingCount: pending,
	}
	if pending == 0 {
		total, pct, passed, err := s.repo.FetchAttemptTotals(ctx, attempt.AttemptID)
		if err != nil {
			return nil, apperrors.Internal(err)
		}
		resp.TotalScore, resp.Percentage, resp.Passed = total, pct, passed
		s.enqueueGapAnalysis(ctx, attempt.AttemptID)
		s.recordLearningEventsAndTrace(ctx, attempt.AttemptID)
	}
	return resp, nil
}

// enqueueGapAnalysis queues a fully-graded attempt for the Capability 3A
// gap-analysis pipeline (ai-service gap_worker). Non-fatal, mirroring the
// exam-parse queue push: grading already succeeded and is persisted --
// analysis can be re-triggered later if Redis is down.
func (s *Service) enqueueGapAnalysis(ctx context.Context, attemptID uuid.UUID) {
	if err := s.redis.LPush(ctx, "queue:gap:analyze", attemptID.String()).Err(); err != nil {
		fmt.Printf("⚠️ Redis queue push failed for gap analysis of attempt %s: %v\n", attemptID, err)
	}
}

// recordLearningEventsAndTrace is EG-GCKT Milestone 1's event-wiring seam
// -- called at the exact point an attempt is confirmed fully graded, same
// as enqueueGapAnalysis. Unlike the Neo4j mirrors and the gap-analysis
// queue push, students.learning_events itself is written synchronously
// and transactionally here (not best-effort/async) because it's the
// record every downstream EG-GCKT engine depends on for correctness --
// see RecordLearningEvents' doc comment. Only the *notification* to
// ai-service (queue:gckt:trace, a separate Redis list from
// queue:gap:analyze so BRPOP on one queue can never starve the other's
// consumer) is best-effort/non-fatal, matching enqueueGapAnalysis.
func (s *Service) recordLearningEventsAndTrace(ctx context.Context, attemptID uuid.UUID) {
	if err := s.repo.RecordLearningEvents(ctx, attemptID); err != nil {
		fmt.Printf("⚠️ Failed to record learning events for attempt %s: %v\n", attemptID, err)
		return
	}
	if err := s.redis.LPush(ctx, "queue:gckt:trace", attemptID.String()).Err(); err != nil {
		fmt.Printf("⚠️ Redis queue push failed for knowledge tracing of attempt %s: %v\n", attemptID, err)
	}
}

// gradeMCQOrPend auto-grades an mcq question against its answer_key when
// present; everything else is left pending (marks_awarded nil).
func gradeMCQOrPend(q repository.QuestionForGrading, response string) repository.GradedAnswer {
	ga := repository.GradedAnswer{
		QuestionID:    q.ID,
		AnswerText:    response,
		MarksPossible: q.Marks,
	}
	if q.QuestionType != "mcq" || q.AnswerKey == nil {
		return ga
	}
	correct, ok := q.AnswerKey["correctOption"]
	if !ok {
		return ga
	}
	isCorrect := strings.EqualFold(strings.TrimSpace(response), strings.TrimSpace(correct))
	marks := 0.0
	if isCorrect {
		marks = float64(q.Marks)
	}
	ga.MarksAwarded = &marks
	ga.Passed = &isCorrect
	return ga
}

// BulkGradeExam is Flow 2 (paper/teacher-encoded): the teacher already
// graded on paper and is typing in the results, not triggering a
// re-grade. entry.Value is an MCQ option letter (cross-checked against
// answer_key if present, purely for the passed flag) or the teacher's own
// numeric marks for everything else.
func (s *Service) BulkGradeExam(ctx context.Context, examID, gradedBy uuid.UUID, req dto.BulkGradeRequest) (*dto.BulkGradeResponse, error) {
	// gradedBy doubles as the caller for the ownership check -- it's
	// already the JWT-derived teacher/school_admin userID (see
	// handler.BulkGradeExam), no separate param needed.
	if err := s.verifyCallerOwnsExam(ctx, gradedBy, examID); err != nil {
		return nil, err
	}
	exam, err := s.repo.FetchExamForValidation(ctx, examID)
	if errors.Is(err, repository.ErrNotFound) {
		return nil, apperrors.NotFound("exam not found")
	}
	if err != nil {
		return nil, apperrors.Internal(err)
	}

	questions, err := s.repo.FetchQuestionsForGrading(ctx, examID)
	if err != nil {
		return nil, apperrors.Internal(err)
	}
	questionsByID := make(map[uuid.UUID]repository.QuestionForGrading, len(questions))
	for _, q := range questions {
		questionsByID[q.ID] = q
	}

	byStudent := make(map[uuid.UUID][]dto.GradeEntry)
	for _, e := range req.Entries {
		if _, ok := questionsByID[e.QuestionID]; !ok {
			return nil, apperrors.BadRequest(fmt.Sprintf("question %s is not part of this exam", e.QuestionID))
		}
		byStudent[e.StudentID] = append(byStudent[e.StudentID], e)
	}

	attemptsTouched, answersSaved := 0, 0
	for studentID, entries := range byStudent {
		attemptID, err := s.repo.FindAttempt(ctx, studentID, examID)
		if err != nil {
			return nil, apperrors.Internal(err)
		}
		if attemptID == nil {
			id, err := s.repo.CreateAttempt(ctx, studentID, examID, exam.SchoolID, true)
			if err != nil {
				return nil, apperrors.Internal(err)
			}
			attemptID = &id
		}

		answers := make([]repository.GradedAnswer, 0, len(entries))
		for _, e := range entries {
			answers = append(answers, gradeTeacherEntry(questionsByID[e.QuestionID], e.Value))
		}
		if err := s.repo.SaveStudentAnswers(ctx, *attemptID, studentID, exam.SchoolID, answers, &gradedBy); err != nil {
			return nil, apperrors.Internal(err)
		}
		if err := s.repo.RecomputeAttemptTotals(ctx, *attemptID); err != nil {
			return nil, apperrors.Internal(err)
		}
		// A bulk-grade entry may have just finished the attempt's last
		// pending answer -- only a finalized attempt (totals written) is
		// worth analyzing.
		if total, _, _, err := s.repo.FetchAttemptTotals(ctx, *attemptID); err == nil && total != nil {
			s.enqueueGapAnalysis(ctx, *attemptID)
			s.recordLearningEventsAndTrace(ctx, *attemptID)
		}
		attemptsTouched++
		answersSaved += len(answers)
	}

	return &dto.BulkGradeResponse{AttemptsTouched: attemptsTouched, AnswersSaved: answersSaved}, nil
}

// gradeTeacherEntry: the teacher already graded this by hand -- for mcq,
// Value is the option letter they circled (cross-checked against
// answer_key when available, only to set Passed; a mismatch doesn't
// override their entry). For everything else, Value is their marks
// directly. Numeric input always wins, for any question type: if a teacher
// types a mark for an MCQ question (common when there's no parsed
// answer_key to auto-derive correctness from a letter), that's trusted
// as-is rather than forced through letter interpretation. Only when the
// input isn't numeric do we fall back to treating it as an MCQ option
// letter, cross-checked against answer_key when one exists.
func gradeTeacherEntry(q repository.QuestionForGrading, value string) repository.GradedAnswer {
	ga := repository.GradedAnswer{QuestionID: q.ID, AnswerText: value, MarksPossible: q.Marks}
	trimmed := strings.TrimSpace(value)

	if marks, err := strconv.ParseFloat(trimmed, 64); err == nil {
		fullCredit := marks >= float64(q.Marks)
		ga.MarksAwarded = &marks
		ga.Passed = &fullCredit
		return ga
	}

	if q.QuestionType == "mcq" {
		marks := 0.0
		if correct, ok := q.AnswerKey["correctOption"]; ok {
			isCorrect := strings.EqualFold(trimmed, strings.TrimSpace(correct))
			if isCorrect {
				marks = float64(q.Marks)
			}
			ga.Passed = &isCorrect
		}
		ga.MarksAwarded = &marks
		return ga
	}

	// Non-mcq, non-numeric input -- can't grade this, leave pending
	// (marks_awarded nil) rather than silently record a wrong score.
	return ga
}
