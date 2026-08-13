package service

import (
	"bytes"
	"context"
	"html/template"

	"github.com/google/uuid"

	"github.com/edugraph-ai/edugraph/internal/assessment/dto"
	apperrors "github.com/edugraph-ai/edugraph/pkg/errors"
)

// GeneratePrintableExam renders Capability 2.2's "Mode B: Print & Paper
// Exam" -- a formatted exam sheet as print-ready HTML rather than a
// server-generated PDF (see printExamTemplate's doc comment for why).
// Teachers open this in a browser tab and use its own "Print / Save as
// PDF" (window.print()), which every browser already does reliably with
// correct fonts/pagination, instead of this service carrying a PDF-layout
// dependency of its own.
func (s *Service) GeneratePrintableExam(ctx context.Context, userID, examID uuid.UUID) (string, error) {
	exam, err := s.GetExam(ctx, userID, examID)
	if err != nil {
		return "", err
	}
	questions, err := s.repo.FetchQuestionsForStudent(ctx, examID)
	if err != nil {
		return "", apperrors.Internal(err)
	}
	if len(questions) == 0 {
		return "", apperrors.BadRequest("exam has no parsed questions yet -- wait for parsing to finish")
	}

	return renderTemplate(printExamTemplate, examPrintView{
		Exam:      exam,
		Questions: questions,
	})
}

// GenerateAnswerKeySheet renders the companion "optical answer key" from
// the same PRD requirement -- a bubble-style reference sheet marking the
// correct option for every MCQ question, for a teacher to grade
// hand-collected paper answer sheets against (or, on the digital side, to
// cross-check the auto-grader). Non-MCQ questions are listed with a
// "graded manually" note since there's no bubble to mark.
func (s *Service) GenerateAnswerKeySheet(ctx context.Context, userID, examID uuid.UUID) (string, error) {
	exam, err := s.GetExam(ctx, userID, examID)
	if err != nil {
		return "", err
	}
	questions, err := s.repo.FetchQuestionsForGrading(ctx, examID)
	if err != nil {
		return "", apperrors.Internal(err)
	}
	if len(questions) == 0 {
		return "", apperrors.BadRequest("exam has no parsed questions yet -- wait for parsing to finish")
	}

	rows := make([]answerKeyRow, 0, len(questions))
	for _, q := range questions {
		row := answerKeyRow{
			SequenceNumber: q.SequenceNumber,
			QuestionType:   q.QuestionType,
			Marks:          q.Marks,
			Options:        q.Options,
		}
		if q.QuestionType == "mcq" {
			row.CorrectOption = q.AnswerKey["correctOption"]
		}
		rows = append(rows, row)
	}

	return renderTemplate(answerKeyTemplate, answerKeyView{
		Exam: exam,
		Rows: rows,
	})
}

type examPrintView struct {
	Exam      *dto.ExamStatus
	Questions []dto.StudentQuestion
}

type answerKeyRow struct {
	SequenceNumber int
	QuestionType   string
	Marks          int
	Options        []dto.QuestionOption
	CorrectOption  string // "" for non-mcq
}

type answerKeyView struct {
	Exam *dto.ExamStatus
	Rows []answerKeyRow
}

func renderTemplate(tmpl *template.Template, data any) (string, error) {
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", apperrors.Internal(err)
	}
	return buf.String(), nil
}
