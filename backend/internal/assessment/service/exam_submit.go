package service

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"

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

// ListExamQuestionsForStudent is what the exam-taking page fetches before
// the student answers anything -- dto.StudentQuestion has no
// answer_key/clo_code fields, unlike the grading-side QuestionForGrading.
func (s *Service) ListExamQuestionsForStudent(ctx context.Context, userID, examID uuid.UUID) ([]dto.StudentQuestion, error) {
	_, _, err := s.verifyStudentAccess(ctx, userID, examID)
	if err != nil {
		return nil, err
	}
	questions, err := s.repo.FetchQuestionsForStudent(ctx, examID)
	if err != nil {
		return nil, apperrors.Internal(err)
	}
	return questions, nil
}

// ListQuestionsForGrading backs the teacher-facing "Grade Exam" spreadsheet's
// column headers -- no per-teacher-school ownership check, matching the
// same convention already used by GetExam/ValidateExam/PublishExam (role
// gating alone, enforced by middleware.RequireRole in the router).
func (s *Service) ListQuestionsForGrading(ctx context.Context, examID uuid.UUID) ([]dto.GradingQuestion, error) {
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
func (s *Service) SubmitExam(ctx context.Context, userID, examID uuid.UUID, req dto.SubmitExamRequest) (*dto.SubmitExamResponse, error) {
	_, student, err := s.verifyStudentAccess(ctx, userID, examID)
	if err != nil {
		return nil, err
	}

	existing, err := s.repo.FindAttempt(ctx, student.ID, examID)
	if err != nil {
		return nil, apperrors.Internal(err)
	}
	if existing != nil {
		return nil, apperrors.Conflict("you have already submitted this exam")
	}

	questions, err := s.repo.FetchQuestionsForGrading(ctx, examID)
	if err != nil {
		return nil, apperrors.Internal(err)
	}
	questionsByID := make(map[uuid.UUID]repository.QuestionForGrading, len(questions))
	for _, q := range questions {
		questionsByID[q.ID] = q
	}

	attemptID, err := s.repo.CreateAttempt(ctx, student.ID, examID, student.SchoolID, false)
	if err != nil {
		return nil, apperrors.Internal(err)
	}

	answers := make([]repository.GradedAnswer, 0, len(req.Answers))
	graded, pending := 0, 0
	for _, a := range req.Answers {
		q, ok := questionsByID[a.QuestionID]
		if !ok {
			return nil, apperrors.BadRequest(fmt.Sprintf("question %s is not part of this exam", a.QuestionID))
		}
		ga := gradeMCQOrPend(q, a.Response)
		ga.TimeSpentSecs = a.TimeSpentSecs
		answers = append(answers, ga)
		if ga.MarksAwarded != nil {
			graded++
		} else {
			pending++
		}
	}

	if err := s.repo.SaveStudentAnswers(ctx, attemptID, student.ID, student.SchoolID, answers); err != nil {
		return nil, apperrors.Internal(err)
	}
	if err := s.repo.RecomputeAttemptTotals(ctx, attemptID); err != nil {
		return nil, apperrors.Internal(err)
	}

	resp := &dto.SubmitExamResponse{
		AttemptID:           attemptID,
		GradedCount:         graded,
		PendingGradingCount: pending,
	}
	if pending == 0 {
		total, pct, passed, err := s.repo.FetchAttemptTotals(ctx, attemptID)
		if err != nil {
			return nil, apperrors.Internal(err)
		}
		resp.TotalScore, resp.Percentage, resp.Passed = total, pct, passed
		s.enqueueGapAnalysis(ctx, attemptID)
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
func (s *Service) BulkGradeExam(ctx context.Context, examID uuid.UUID, req dto.BulkGradeRequest) (*dto.BulkGradeResponse, error) {
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
		if err := s.repo.SaveStudentAnswers(ctx, *attemptID, studentID, exam.SchoolID, answers); err != nil {
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
