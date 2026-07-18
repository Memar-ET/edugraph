package service

import (
	"context"
	"encoding/json"
	"errors"
	"math"
	"sort"
	"time"

	"github.com/google/uuid"

	"github.com/edugraph-ai/edugraph/internal/assessment/dto"
	"github.com/edugraph-ai/edugraph/internal/assessment/repository"
	apperrors "github.com/edugraph-ai/edugraph/pkg/errors"
)

// Capability 4B: Exam Quality Scoring. Thresholds:
const (
	// mastery_records confidence at/above which a student counts as
	// having mastered the question's topic (same 0.5 line as 3A's
	// WEAK_THRESHOLD and the attempt pass mark).
	masteryThreshold = 0.5
	// Actual-difficulty buckets from mean score ratio.
	easyRatio = 0.75
	hardRatio = 0.40
	// Time anomaly: a question needs this many timed answers to be
	// judged, and is flagged when its average falls below
	// timeAnomalyFraction of the exam's median question time (with an
	// absolute floor -- nothing real takes under a few seconds).
	minTimedAnswers      = 3
	timeAnomalyFraction  = 0.3
	timeAnomalyFloorSecs = 10.0
)

// GetExamQuality computes the 4B report fresh on every read (one exam's
// answers -- cheap), persists it for 4C's school-level aggregation, and
// writes performance-derived difficulty back to
// questions.calibrated_difficulty. Caller must belong to the exam's
// school -- exam quality is teacher/school-facing, not public.
func (s *Service) GetExamQuality(ctx context.Context, userID, examID uuid.UUID) (*dto.ExamQualityResponse, error) {
	exam, err := s.repo.FetchExamForQuality(ctx, examID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, apperrors.NotFound("exam not found")
		}
		return nil, apperrors.Internal(err)
	}
	schoolID, err := s.repo.TeacherSchoolID(ctx, userID)
	if err != nil {
		return nil, apperrors.Internal(err)
	}
	if schoolID != exam.SchoolID {
		return nil, apperrors.Forbidden("exam belongs to a different school")
	}

	resp, discAvg, calibrated, err := s.computeExamQuality(ctx, exam)
	if err != nil {
		return nil, err
	}

	// Best-effort side effects: calibration write-back + persisted
	// report. The report itself is already in hand -- failing to store
	// it must not fail the read.
	_ = s.repo.WriteCalibratedDifficulties(ctx, calibrated)
	if raw, jerr := json.Marshal(resp); jerr == nil {
		_ = s.repo.UpsertExamQualityReport(ctx, exam.ID, exam.SchoolID,
			resp.AttemptsAnalyzed, discAvg, resp.QualityScore, raw)
	}
	return resp, nil
}

// EnsureExamQualityReport is 4C's entry point: compute + persist the
// report for an exam without the caller-ownership check (the nightly
// worker owns no school). Skips nothing -- exams with zero graded
// answers still get a report row (attempts 0, nil discrimination).
func (s *Service) EnsureExamQualityReport(ctx context.Context, examID uuid.UUID) error {
	exam, err := s.repo.FetchExamForQuality(ctx, examID)
	if err != nil {
		return err
	}
	resp, discAvg, calibrated, err := s.computeExamQuality(ctx, exam)
	if err != nil {
		return err
	}
	_ = s.repo.WriteCalibratedDifficulties(ctx, calibrated)
	raw, err := json.Marshal(resp)
	if err != nil {
		return err
	}
	return s.repo.UpsertExamQualityReport(ctx, exam.ID, exam.SchoolID,
		resp.AttemptsAnalyzed, discAvg, resp.QualityScore, raw)
}

func (s *Service) computeExamQuality(ctx context.Context, exam *repository.ExamForQuality) (*dto.ExamQualityResponse, *float64, map[uuid.UUID]string, error) {
	questions, err := s.repo.FetchQuestionsForQuality(ctx, exam.ID)
	if err != nil {
		return nil, nil, nil, apperrors.Internal(err)
	}
	answers, err := s.repo.FetchAnswersForQuality(ctx, exam.ID)
	if err != nil {
		return nil, nil, nil, apperrors.Internal(err)
	}
	attempts, err := s.repo.FinalizedAttemptCount(ctx, exam.ID)
	if err != nil {
		return nil, nil, nil, apperrors.Internal(err)
	}
	mandatory, err := s.repo.FetchMandatoryCLOs(ctx, exam.SubjectCode, exam.GradeLevel)
	if err != nil {
		return nil, nil, nil, apperrors.Internal(err)
	}

	byQuestion := make(map[uuid.UUID][]repository.AnswerForQuality, len(questions))
	for _, a := range answers {
		byQuestion[a.QuestionID] = append(byQuestion[a.QuestionID], a)
	}

	calibrated := make(map[uuid.UUID]string)
	qualities := make([]dto.QuestionQuality, 0, len(questions))
	var discSum float64
	var discCount int
	var calibAccurate, calibMismatch int

	// Metric 1+2 pass: per-question discrimination + difficulty.
	for _, q := range questions {
		qa := byQuestion[q.ID]
		qq := dto.QuestionQuality{
			QuestionID:     q.ID.String(),
			SequenceNumber: q.SequenceNumber,
			QuestionType:   q.QuestionType,
			GradedAnswers:  len(qa),
		}
		if q.CLOCode != nil {
			qq.CLOCode = *q.CLOCode
		}
		if q.StatedDifficulty != nil {
			qq.StatedDifficulty = *q.StatedDifficulty
		}

		if len(qa) > 0 {
			var ratioSum float64
			for _, a := range qa {
				ratioSum += a.ScoreRatio
			}
			avgRatio := ratioSum / float64(len(qa))
			qq.AvgScoreRatio = math.Round(avgRatio*1000) / 1000
			qq.ActualDifficulty = bucketDifficulty(avgRatio)
			calibrated[q.ID] = qq.ActualDifficulty

			switch qq.StatedDifficulty {
			case "":
				qq.Calibration = "unrated"
			case qq.ActualDifficulty:
				qq.Calibration = "accurate"
				calibAccurate++
			default:
				qq.Calibration = "mismatch"
				calibMismatch++
			}

			if d, method := discriminationIndex(qa); d != nil {
				rounded := math.Round(*d*1000) / 1000
				qq.DiscriminationIndex = &rounded
				qq.DiscriminationMethod = method
				discSum += *d
				discCount++
			}
		} else {
			qq.Calibration = "unrated"
		}

		qualities = append(qualities, qq)
	}

	// Metric 3: time anomalies -- relative to the exam's own median so a
	// generally-fast quiz doesn't flag everything.
	timeAnalysis := applyTimeAnomalies(qualities, byQuestion, questions)

	// Metric 4: mandatory-CLO coverage after the exam.
	coverage := cloCoverage(mandatory, questions, byQuestion)

	var discAvg *float64
	if discCount > 0 {
		v := math.Round(discSum/float64(discCount)*1000) / 1000
		discAvg = &v
	}

	resp := &dto.ExamQualityResponse{
		ExamID:            exam.ID.String(),
		Title:             exam.Title,
		AttemptsAnalyzed:  attempts,
		DiscriminationAvg: discAvg,
		QualityScore:      examQualityScore(discAvg, calibAccurate, calibMismatch, coverage),
		Questions:         qualities,
		CLOCoverage:       coverage,
		TimeAnalysis:      timeAnalysis,
		ComputedAt:        time.Now().UTC(),
	}
	return resp, discAvg, calibrated, nil
}

func bucketDifficulty(avgRatio float64) string {
	switch {
	case avgRatio >= easyRatio:
		return "easy"
	case avgRatio < hardRatio:
		return "hard"
	default:
		return "medium"
	}
}

// discriminationIndex prefers the mastery split ("do students who
// mastered the topic actually pass?"): pass rate among students with
// topic mastery >= threshold minus pass rate among those below it,
// counting only students with mastery EVIDENCE. When either group is
// empty it falls back to the classic index: top half vs bottom half of
// answerers ranked by their overall attempt percentage. Returns nil when
// no split yields two non-empty groups (e.g. a single student).
func discriminationIndex(qa []repository.AnswerForQuality) (*float64, string) {
	var masteredPass, masteredTotal, weakPass, weakTotal int
	for _, a := range qa {
		if a.TopicMastery == nil {
			continue
		}
		if *a.TopicMastery >= masteryThreshold {
			masteredTotal++
			if a.Passed {
				masteredPass++
			}
		} else {
			weakTotal++
			if a.Passed {
				weakPass++
			}
		}
	}
	if masteredTotal > 0 && weakTotal > 0 {
		d := float64(masteredPass)/float64(masteredTotal) - float64(weakPass)/float64(weakTotal)
		return &d, "mastery"
	}

	ranked := make([]repository.AnswerForQuality, 0, len(qa))
	for _, a := range qa {
		if a.AttemptPct != nil {
			ranked = append(ranked, a)
		}
	}
	if len(ranked) < 2 {
		return nil, ""
	}
	sort.Slice(ranked, func(i, j int) bool { return *ranked[i].AttemptPct > *ranked[j].AttemptPct })
	half := len(ranked) / 2
	upper, lower := ranked[:half], ranked[len(ranked)-half:]
	passRate := func(group []repository.AnswerForQuality) float64 {
		pass := 0
		for _, a := range group {
			if a.Passed {
				pass++
			}
		}
		return float64(pass) / float64(len(group))
	}
	d := passRate(upper) - passRate(lower)
	return &d, "score_split"
}

// applyTimeAnomalies mutates qualities in place (AvgTimeSecs +
// TimeAnomaly) and returns the exam-level summary.
func applyTimeAnomalies(qualities []dto.QuestionQuality, byQuestion map[uuid.UUID][]repository.AnswerForQuality, questions []repository.QuestionForQuality) dto.TimeAnalysisQuality {
	avgTimes := make(map[uuid.UUID]float64)
	var allAvgs []float64
	for _, q := range questions {
		var sum, n float64
		for _, a := range byQuestion[q.ID] {
			if a.TimeSpentSecs != nil {
				sum += float64(*a.TimeSpentSecs)
				n++
			}
		}
		if n >= minTimedAnswers {
			avg := sum / n
			avgTimes[q.ID] = avg
			allAvgs = append(allAvgs, avg)
		}
	}

	analysis := dto.TimeAnalysisQuality{QuestionsWithTiming: len(allAvgs)}
	// One timed question has no cohort to be anomalous against.
	if len(allAvgs) < 2 {
		return analysis
	}
	sort.Float64s(allAvgs)
	median := allAvgs[len(allAvgs)/2]
	if len(allAvgs)%2 == 0 {
		median = (allAvgs[len(allAvgs)/2-1] + allAvgs[len(allAvgs)/2]) / 2
	}
	rounded := math.Round(median*10) / 10
	analysis.MedianAvgSecs = &rounded
	analysis.Evaluated = true

	threshold := math.Max(timeAnomalyFloorSecs, timeAnomalyFraction*median)
	for i := range qualities {
		qid, err := uuid.Parse(qualities[i].QuestionID)
		if err != nil {
			continue
		}
		if avg, ok := avgTimes[qid]; ok {
			r := math.Round(avg*10) / 10
			qualities[i].AvgTimeSecs = &r
			qualities[i].TimeAnomaly = avg < threshold
		}
	}
	return analysis
}

func cloCoverage(mandatory []repository.MandatoryCLO, questions []repository.QuestionForQuality, byQuestion map[uuid.UUID][]repository.AnswerForQuality) dto.CLOCoverageQuality {
	descByCode := make(map[string]string, len(mandatory))
	for _, c := range mandatory {
		descByCode[c.Code] = c.Description
	}

	tested := make(map[string]bool)     // clo -> tested by >=1 question
	anyCorrect := make(map[string]bool) // clo -> >=1 passing graded answer
	graded := make(map[string]bool)     // clo -> >=1 graded answer at all
	for _, q := range questions {
		if q.CLOCode == nil {
			continue
		}
		code := *q.CLOCode
		tested[code] = true
		for _, a := range byQuestion[q.ID] {
			graded[code] = true
			if a.Passed {
				anyCorrect[code] = true
			}
		}
	}

	cov := dto.CLOCoverageQuality{
		MandatoryTotal: len(mandatory),
		ZeroCorrect:    []dto.CLOFlag{},
		Untested:       []dto.CLOFlag{},
	}
	for _, c := range mandatory {
		if !tested[c.Code] {
			cov.Untested = append(cov.Untested, dto.CLOFlag{CLOCode: c.Code, Description: c.Description})
			continue
		}
		cov.MandatoryTested++
		// "Zero students answered correctly" requires evidence someone
		// tried -- an ungraded CLO isn't a failed CLO.
		if graded[c.Code] && !anyCorrect[c.Code] {
			cov.ZeroCorrect = append(cov.ZeroCorrect, dto.CLOFlag{CLOCode: c.Code, Description: c.Description})
		}
	}
	return cov
}

// examQualityScore folds the three exam-level signals into 0..100:
// 50% discrimination (normalized from -1..1), 20% difficulty-calibration
// accuracy, 30% mandatory-CLO coverage. Missing signals score neutral
// (0.5) rather than punishing exams that predate timing/difficulty data.
func examQualityScore(discAvg *float64, calibAccurate, calibMismatch int, cov dto.CLOCoverageQuality) float64 {
	disc := 0.5
	if discAvg != nil {
		disc = (*discAvg + 1) / 2
	}
	calib := 0.5
	if calibAccurate+calibMismatch > 0 {
		calib = float64(calibAccurate) / float64(calibAccurate+calibMismatch)
	}
	coverage := 1.0
	if cov.MandatoryTotal > 0 {
		coverage = float64(cov.MandatoryTested) / float64(cov.MandatoryTotal)
	}
	score := 100 * (0.5*disc + 0.2*calib + 0.3*coverage)
	return math.Round(score*10) / 10
}
