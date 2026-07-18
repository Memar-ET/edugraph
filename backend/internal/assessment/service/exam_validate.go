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
func (s *Service) ValidateExam(ctx context.Context, examID uuid.UUID) (*dto.ValidationReport, error) {
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

	return report, nil
}

// PublishExam only succeeds once the exam has been validated at least once
// (status 'validation_pending'). Not blocked by report content -- a
// warning system, not a hard quality gate; the teacher decides.
func (s *Service) PublishExam(ctx context.Context, examID uuid.UUID) (*dto.PublishResponse, error) {
	if err := s.repo.PublishExam(ctx, examID); err != nil {
		if errors.Is(err, repository.ErrNotValidated) {
			return nil, apperrors.Conflict("exam must be validated (POST .../validate) before it can be published")
		}
		return nil, apperrors.Internal(err)
	}
	return &dto.PublishResponse{ExamID: examID, Status: "published"}, nil
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
