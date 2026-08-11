package service

import (
	"context"
	"encoding/json"
	"errors"
	"math"
	"time"

	"github.com/google/uuid"

	"github.com/edugraph-ai/edugraph/internal/assessment/dto"
	"github.com/edugraph-ai/edugraph/internal/assessment/repository"
	apperrors "github.com/edugraph-ai/edugraph/pkg/errors"
)

// Capability 4C: School Quality Scoring.
const (
	// Impl-plan weights: 0.30 CLO coverage, 0.25 student mastery,
	// 0.25 exam quality, 0.20 curriculum compliance.
	wCLOCoverage = 0.30
	wMastery     = 0.25
	wExamQuality = 0.25
	wCompliance  = 0.20

	// Below this curriculum compliance the school is flagged for
	// Ministry review -- mandatory sections look skipped.
	complianceReviewThreshold = 50.0

	schoolQualityCacheTTL = time.Hour
)

func schoolQualityCacheKey(schoolID uuid.UUID) string {
	return "school_quality:" + schoolID.String()
}

// ComputeSchoolQuality recalculates one (school, subject, grade) combo:
// backfills missing 4B reports first (their discrimination average is a
// component), applies the weighted formula, persists, invalidates the
// dashboard cache, and notifies the Ministry when the combo NEWLY drops
// below the compliance threshold.
func (s *Service) ComputeSchoolQuality(ctx context.Context, c repository.SchoolSubjectGrade) (*dto.SchoolQualityScore, error) {
	missing, err := s.repo.PublishedExamsMissingQuality(ctx, c)
	if err != nil {
		return nil, err
	}
	for _, examID := range missing {
		// Best-effort: one uncomputable exam shouldn't sink the whole
		// school score -- it simply won't contribute a discrimination avg.
		_ = s.EnsureExamQualityReport(ctx, examID)
	}

	cloTotal, cloTested, err := s.repo.CLOCoverageStats(ctx, c)
	if err != nil {
		return nil, err
	}
	students, mastered, err := s.repo.StudentMasteryStats(ctx, c)
	if err != nil {
		return nil, err
	}
	discAvg, err := s.repo.ExamDiscriminationAvg(ctx, c)
	if err != nil {
		return nil, err
	}
	topicsTotal, topicsAssessed, err := s.repo.ComplianceStats(ctx, c)
	if err != nil {
		return nil, err
	}

	cloCoverage := pct(cloTested, cloTotal)
	masteryPct := pct(mastered, students)
	// Discrimination is -1..1; normalize to 0..100. No exam data at all
	// scores 0 -- the formula rewards hard evidence, not its absence.
	examQuality := 0.0
	if discAvg != nil {
		examQuality = math.Round((*discAvg+1)/2*1000) / 10
	}
	compliance := pct(topicsAssessed, topicsTotal)

	composite := wCLOCoverage*cloCoverage + wMastery*masteryPct +
		wExamQuality*examQuality + wCompliance*compliance
	composite = math.Round(composite*10) / 10

	row := repository.QualityScoreRow{
		SubjectCode:          c.SubjectCode,
		GradeLevel:           c.GradeLevel,
		CLOCoveragePct:       cloCoverage,
		StudentMasteryPct:    masteryPct,
		ExamQualityAvg:       examQuality,
		CurriculumCompliance: compliance,
		CompositeScore:       composite,
		FlaggedForReview:     compliance < complianceReviewThreshold,
	}
	newlyFlagged, err := s.repo.UpsertQualityScore(ctx, c, row)
	if err != nil {
		return nil, err
	}
	if newlyFlagged {
		// Notification failure is not a scoring failure.
		_ = s.repo.NotifyMinistryOfFlag(ctx, c, compliance)
	}

	// The nightly write invalidates the hour cache rather than racing it.
	s.redis.Del(ctx, schoolQualityCacheKey(c.SchoolID))

	return &dto.SchoolQualityScore{
		SubjectCode:          row.SubjectCode,
		GradeLevel:           row.GradeLevel,
		CLOCoveragePct:       row.CLOCoveragePct,
		StudentMasteryPct:    row.StudentMasteryPct,
		ExamQualityAvg:       row.ExamQualityAvg,
		CurriculumCompliance: row.CurriculumCompliance,
		CompositeScore:       row.CompositeScore,
		FlaggedForReview:     row.FlaggedForReview,
		ComputedAt:           time.Now().UTC(),
	}, nil
}

// RecomputeAllSchoolQuality is the nightly batch: every combo with a
// published exam. Returns how many combos were scored.
func (s *Service) RecomputeAllSchoolQuality(ctx context.Context) (int, error) {
	combos, err := s.repo.QualityCombos(ctx)
	if err != nil {
		return 0, err
	}
	scored := 0
	for _, c := range combos {
		if _, err := s.ComputeSchoolQuality(ctx, c); err == nil {
			scored++
		}
	}
	return scored, nil
}

// GetSchoolQualityScores serves the dashboard read: Redis 1h cache ->
// Postgres -> compute-on-demand for a school nobody has scored yet.
// school_admins are pinned to their own school; regional/ministry admins
// may read any school (role enforcement beyond that is the router's).
func (s *Service) GetSchoolQualityScores(ctx context.Context, userID uuid.UUID, role string, schoolID uuid.UUID) (*dto.SchoolQualityResponse, error) {
	if role == "school_admin" {
		own, err := s.repo.TeacherSchoolID(ctx, userID)
		if err != nil {
			if errors.Is(err, repository.ErrNotFound) {
				return nil, apperrors.NotFound("user not found")
			}
			return nil, apperrors.Internal(err)
		}
		if own != schoolID {
			return nil, apperrors.Forbidden("school admins can only view their own school")
		}
	}

	cacheKey := schoolQualityCacheKey(schoolID)
	if cached, err := s.redis.Get(ctx, cacheKey).Bytes(); err == nil {
		var resp dto.SchoolQualityResponse
		if json.Unmarshal(cached, &resp) == nil {
			resp.Source = "redis"
			return &resp, nil
		}
	}

	rows, err := s.repo.ListQualityScores(ctx, schoolID)
	if err != nil {
		return nil, apperrors.Internal(err)
	}
	if len(rows) == 0 {
		// First read before any nightly run: compute this school's
		// combos on demand so the dashboard is never empty when exam
		// data exists.
		combos, err := s.repo.QualityCombos(ctx)
		if err != nil {
			return nil, apperrors.Internal(err)
		}
		for _, c := range combos {
			if c.SchoolID == schoolID {
				_, _ = s.ComputeSchoolQuality(ctx, c)
			}
		}
		if rows, err = s.repo.ListQualityScores(ctx, schoolID); err != nil {
			return nil, apperrors.Internal(err)
		}
	}

	resp := &dto.SchoolQualityResponse{
		SchoolID: schoolID.String(),
		Source:   "postgres",
		Scores:   make([]dto.SchoolQualityScore, 0, len(rows)),
	}
	for _, q := range rows {
		resp.Scores = append(resp.Scores, dto.SchoolQualityScore{
			SubjectCode:          q.SubjectCode,
			GradeLevel:           q.GradeLevel,
			CLOCoveragePct:       q.CLOCoveragePct,
			StudentMasteryPct:    q.StudentMasteryPct,
			ExamQualityAvg:       q.ExamQualityAvg,
			CurriculumCompliance: q.CurriculumCompliance,
			CompositeScore:       q.CompositeScore,
			FlaggedForReview:     q.FlaggedForReview,
			ComputedAt:           q.ComputedAt,
		})
	}

	if raw, err := json.Marshal(resp); err == nil {
		s.redis.Set(ctx, cacheKey, raw, schoolQualityCacheTTL)
	}
	return resp, nil
}

func pct(part, whole int) float64 {
	if whole == 0 {
		return 0
	}
	return math.Round(float64(part)/float64(whole)*1000) / 10
}
