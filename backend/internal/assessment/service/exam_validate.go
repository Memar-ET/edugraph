package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/edugraph-ai/edugraph/internal/assessment/dto"
	"github.com/edugraph-ai/edugraph/internal/assessment/repository"
	apperrors "github.com/edugraph-ai/edugraph/pkg/errors"
)

// minHigherOrderBloomPct mirrors the spec's literal example: "Ministry
// requires at least 30% Analyzing or Applying".
const minHigherOrderBloomPct = 30.0

// maxHardPercentByScope: escalating stakes/comprehensiveness -- a Unit
// Test has the least room for Hard questions, a Final Exam the most.
var maxHardPercentByScope = map[string]float64{
	"unit_test":  20.0,
	"midterm":    30.0,
	"final_exam": 40.0,
}

// bloomToDifficulty: assessment.questions.difficulty_level is never
// populated by 2A's parser, so difficulty is derived from the linked CLO's
// Bloom level via this standard curriculum-design mapping.
var bloomToDifficulty = map[string]string{
	"remember":   "easy",
	"understand": "easy",
	"apply":      "medium",
	"analyse":    "medium",
	"evaluate":   "hard",
	"create":     "hard",
}

var bloomLevels = []string{"remember", "understand", "apply", "analyse", "evaluate", "create"}

// ValidateExam computes Capability 2B's 5-part report and stores it,
// moving the exam to 'validation_pending'.
func (s *Service) ValidateExam(ctx context.Context, userID, examID uuid.UUID) (*dto.ValidationReport, error) {
	if err := s.verifyCallerOwnsExam(ctx, userID, examID); err != nil {
		return nil, err
	}
	exam, err := s.repo.FetchExamForValidation(ctx, examID)
	if errors.Is(err, repository.ErrNotFound) {
		return nil, apperrors.NotFound("exam not found")
	}
	if err != nil {
		return nil, apperrors.Internal(err)
	}

	questions, err := s.repo.FetchQuestionsWithClo(ctx, examID)
	if err != nil {
		return nil, apperrors.Internal(err)
	}
	if len(questions) == 0 {
		return nil, apperrors.BadRequest("exam has no parsed questions yet -- wait for parsing to finish")
	}

	closInScope, err := s.repo.FetchCurriculumCLOsInScope(ctx, exam.SubjectCode, exam.GradeLevel, exam.UnitNumbers)
	if err != nil {
		return nil, apperrors.Internal(err)
	}

	topicsInScope, err := s.repo.FetchTopicsInScope(ctx, exam.SubjectCode, exam.GradeLevel, exam.UnitNumbers)
	if err != nil {
		return nil, apperrors.Internal(err)
	}

	topicIDs := distinctTopicIDs(topicsInScope)
	prereqWarnings, err := s.repo.FetchPrerequisiteWarnings(ctx, topicIDs)
	if err != nil {
		return nil, apperrors.Internal(err)
	}

	report := &dto.ValidationReport{
		GeneratedAt:            time.Now().UTC(),
		Scope:                  describeScope(exam.ExamScope, exam.UnitNumbers),
		CLOCoverage:            computeCLOCoverage(questions, closInScope),
		BloomBalance:           computeBloomBalance(questions),
		DifficultyDistribution: computeDifficultyDistribution(questions, exam.ExamScope),
		TopicCoverage:          computeTopicCoverage(questions, topicsInScope),
		PrerequisiteWarnings:   toPrerequisiteWarningEntries(prereqWarnings),
	}

	reportJSON, err := json.Marshal(report)
	if err != nil {
		return nil, apperrors.Internal(err)
	}
	if err := s.repo.SaveValidationReport(ctx, examID, reportJSON); err != nil {
		return nil, apperrors.Internal(err)
	}
	s.repo.RecordAuditAction(ctx, "exam.validate", "exam", examID.String())

	return report, nil
}

// PublishExam only succeeds once the exam has been validated at least
// once (status 'validation_pending') AND passes the hard structural
// check below. The curriculum/psychometric report ValidateExam produces
// stays advisory by design (a teacher can publish despite CLO-coverage
// warnings or an unbalanced Bloom distribution -- pedagogical judgment
// calls, not correctness bugs). Structural validity is different: a
// missing answer key or a question with no valid correct option is not
// a judgment call, it's a broken exam that would corrupt every
// student's grade, so it hard-blocks regardless of the soft report.
func (s *Service) PublishExam(ctx context.Context, userID, examID uuid.UUID) (*dto.PublishResponse, error) {
	if err := s.verifyCallerOwnsExam(ctx, userID, examID); err != nil {
		return nil, err
	}

	questions, err := s.repo.FetchQuestionsForGrading(ctx, examID)
	if err != nil {
		return nil, apperrors.Internal(err)
	}
	if problems := validateExamStructure(questions); len(problems) > 0 {
		return nil, apperrors.BadRequest("exam cannot be published: " + strings.Join(problems, "; "))
	}

	if err := s.repo.PublishExam(ctx, examID); err != nil {
		if errors.Is(err, repository.ErrNotValidated) {
			return nil, apperrors.Conflict("exam must be validated (POST .../validate) before it can be published")
		}
		return nil, apperrors.Internal(err)
	}
	s.repo.RecordAuditAction(ctx, "exam.publish", "exam", examID.String())
	return &dto.PublishResponse{ExamID: examID, Status: "published"}, nil
}

// validateExamStructure is the hard, non-negotiable pre-publish gate
// (Part 34): every problem found is reported, not just the first, so a
// teacher can fix everything in one pass instead of one publish-attempt
// per bug.
func validateExamStructure(questions []repository.QuestionForGrading) []string {
	var problems []string
	if len(questions) == 0 {
		return []string{"exam has no questions"}
	}
	for _, q := range questions {
		label := fmt.Sprintf("question %d", q.SequenceNumber)
		if strings.TrimSpace(q.QuestionText) == "" {
			problems = append(problems, label+": question text is empty")
		}
		if q.Marks <= 0 {
			problems = append(problems, label+": marks must be greater than zero")
		}
		if q.QuestionType != "mcq" {
			continue
		}
		if len(q.Options) < 2 {
			problems = append(problems, label+": mcq must have at least 2 options")
			continue
		}
		seenLetters := make(map[string]bool, len(q.Options))
		for _, o := range q.Options {
			if seenLetters[o.Letter] {
				problems = append(problems, fmt.Sprintf("%s: duplicate option letter %q", label, o.Letter))
			}
			seenLetters[o.Letter] = true
		}
		if q.AnswerKey == nil {
			problems = append(problems, label+": mcq has no answer key")
			continue
		}
		correct, ok := q.AnswerKey["correctOption"]
		if !ok || strings.TrimSpace(correct) == "" {
			problems = append(problems, label+": answer key has no correct option set")
			continue
		}
		if !seenLetters[correct] {
			problems = append(problems, fmt.Sprintf("%s: answer key's correct option %q doesn't match any option on this question", label, correct))
		}
	}
	return problems
}

func describeScope(examScope string, unitNumbers []int) string {
	if len(unitNumbers) == 0 {
		if examScope == "final_exam" {
			return fmt.Sprintf("%s: all units", examScope)
		}
		return fmt.Sprintf(
			"%s: unit scope not resolved from the title/document -- checked against the entire subject/grade curriculum as a fallback",
			examScope,
		)
	}
	strs := make([]string, len(unitNumbers))
	for i, n := range unitNumbers {
		strs[i] = strconv.Itoa(n)
	}
	return fmt.Sprintf("%s: units %s", examScope, strings.Join(strs, ", "))
}

func distinctTopicIDs(topics []repository.TopicInScope) []uuid.UUID {
	seen := make(map[uuid.UUID]struct{}, len(topics))
	ids := make([]uuid.UUID, 0, len(topics))
	for _, t := range topics {
		if _, ok := seen[t.ID]; !ok {
			seen[t.ID] = struct{}{}
			ids = append(ids, t.ID)
		}
	}
	return ids
}

func computeCLOCoverage(questions []repository.QuestionWithClo, closInScope []repository.CLOInScope) dto.CLOCoverageReport {
	coveredCodes := make(map[string]struct{})
	for _, q := range questions {
		if q.CloCode != nil {
			coveredCodes[*q.CloCode] = struct{}{}
		}
	}

	var totalMandatory, coveredMandatory, totalClos, coveredClos int
	var missing []string
	for _, c := range closInScope {
		totalClos++
		_, isCovered := coveredCodes[c.Code]
		if isCovered {
			coveredClos++
		}
		if c.IsMandatory {
			totalMandatory++
			if isCovered {
				coveredMandatory++
			} else {
				missing = append(missing, c.Code)
			}
		}
	}
	sort.Strings(missing)

	summary := fmt.Sprintf("Covers %d of %d mandatory CLOs for this scope.", coveredMandatory, totalMandatory)
	if len(missing) > 0 {
		summary += fmt.Sprintf(" Missing: %s.", strings.Join(missing, ", "))
	}

	return dto.CLOCoverageReport{
		TotalMandatoryCLOs:   totalMandatory,
		CoveredMandatoryCLOs: coveredMandatory,
		MissingMandatoryCLOs: missing,
		TotalCLOs:            totalClos,
		CoveredCLOs:          coveredClos,
		Summary:              summary,
	}
}

func computeBloomBalance(questions []repository.QuestionWithClo) dto.BloomBalanceReport {
	counts := map[string]int{}
	unclassified, classified := 0, 0
	for _, q := range questions {
		if q.BloomLevel == nil {
			unclassified++
			continue
		}
		counts[*q.BloomLevel]++
		classified++
	}

	percentages := map[string]float64{}
	var higherOrder float64
	for _, level := range bloomLevels {
		pct := 0.0
		if classified > 0 {
			pct = round2(100 * float64(counts[level]) / float64(classified))
		}
		percentages[level] = pct
		if level == "apply" || level == "analyse" {
			higherOrder += pct
		}
	}
	higherOrder = round2(higherOrder)
	meets := classified > 0 && higherOrder >= minHigherOrderBloomPct

	summary := fmt.Sprintf("%.0f%% of classified questions are Apply/Analyse (Ministry minimum: %.0f%%).", higherOrder, minHigherOrderBloomPct)
	if !meets {
		summary = "Below the Ministry's minimum higher-order-thinking bar -- " + summary
	}
	if unclassified > 0 {
		summary += fmt.Sprintf(" %d question(s) have no matched CLO and were excluded from this calculation.", unclassified)
	}

	return dto.BloomBalanceReport{
		Counts:                  counts,
		Percentages:             percentages,
		UnclassifiedQuestions:   unclassified,
		HigherOrderPercent:      higherOrder,
		MinimumHigherOrderPct:   minHigherOrderBloomPct,
		MeetsMinimumHigherOrder: meets,
		Summary:                 summary,
	}
}

func computeDifficultyDistribution(questions []repository.QuestionWithClo, examScope string) dto.DifficultyDistributionReport {
	counts := map[string]int{}
	unclassified, classified := 0, 0
	for _, q := range questions {
		if q.BloomLevel == nil {
			unclassified++
			continue
		}
		difficulty, ok := bloomToDifficulty[*q.BloomLevel]
		if !ok {
			unclassified++
			continue
		}
		counts[difficulty]++
		classified++
	}

	percentages := map[string]float64{}
	for _, level := range []string{"easy", "medium", "hard"} {
		pct := 0.0
		if classified > 0 {
			pct = round2(100 * float64(counts[level]) / float64(classified))
		}
		percentages[level] = pct
	}
	hardPct := percentages["hard"]
	maxHard, ok := maxHardPercentByScope[examScope]
	if !ok {
		maxHard = 40.0
	}
	exceeds := classified > 0 && hardPct > maxHard

	summary := fmt.Sprintf("%.0f%% Hard questions (recommended max for this scope: %.0f%%).", hardPct, maxHard)
	if exceeds {
		summary = "Too many Hard questions for this exam scope -- " + summary
	}
	if unclassified > 0 {
		summary += fmt.Sprintf(" %d question(s) have no matched CLO and were excluded from this calculation.", unclassified)
	}

	return dto.DifficultyDistributionReport{
		Counts:                counts,
		Percentages:           percentages,
		UnclassifiedQuestions: unclassified,
		HardPercent:           hardPct,
		MaxHardPercentAllowed: maxHard,
		ExceedsMaxHard:        exceeds,
		Summary:               summary,
	}
}

func computeTopicCoverage(questions []repository.QuestionWithClo, topicsInScope []repository.TopicInScope) []dto.TopicCoverageEntry {
	counts := map[uuid.UUID]int{}
	for _, q := range questions {
		if q.TopicID != nil {
			counts[*q.TopicID]++
		}
	}
	entries := make([]dto.TopicCoverageEntry, 0, len(topicsInScope))
	for _, t := range topicsInScope {
		entries = append(entries, dto.TopicCoverageEntry{
			TopicTitle:    t.Title,
			UnitNumber:    t.UnitNumber,
			QuestionCount: counts[t.ID],
		})
	}
	return entries
}

func toPrerequisiteWarningEntries(warnings []repository.PrerequisiteWarning) []dto.PrerequisiteWarningEntry {
	entries := make([]dto.PrerequisiteWarningEntry, 0, len(warnings))
	for _, w := range warnings {
		msg := fmt.Sprintf("%q requires %q", w.TopicTitle, w.PrerequisiteTitle)
		if w.IsCrossGrade {
			msg = fmt.Sprintf("%s (Grade %d) -- ensure students have covered this prerequisite.", msg, w.PrerequisiteGrade)
		} else {
			msg += "."
		}
		entries = append(entries, dto.PrerequisiteWarningEntry{
			TopicTitle:        w.TopicTitle,
			PrerequisiteTitle: w.PrerequisiteTitle,
			PrerequisiteGrade: w.PrerequisiteGrade,
			IsCrossGrade:      w.IsCrossGrade,
			Message:           msg,
		})
	}
	return entries
}

func round2(f float64) float64 {
	return math.Round(f*100) / 100
}
