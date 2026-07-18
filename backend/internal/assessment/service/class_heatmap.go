package service

import (
	"context"
	"errors"
	"fmt"
	"math"

	"github.com/google/uuid"

	"github.com/edugraph-ai/edugraph/internal/assessment/dto"
	"github.com/edugraph-ai/edugraph/internal/assessment/repository"
	apperrors "github.com/edugraph-ai/edugraph/pkg/errors"
)

// alertThresholdPct: above this share of the class struggling with one
// topic, the failure stops looking like individual weakness and starts
// looking like a curriculum-delivery problem, so 4A escalates to the
// cross-grade root-cause traversal (threshold per the impl plan's ">40%").
const alertThresholdPct = 40.0

// GetClassHeatmap answers "which topics is my WHOLE class failing?" for
// the calling teacher's school. Graph-first over the mirrored
// School-ENROLLS->Student-STRUGGLED_WITH->Topic subgraph; falls back to
// students.gap_records when Neo4j is down or hasn't been mirrored yet
// (fresh gap records are synced by the poller, not inline).
func (s *Service) GetClassHeatmap(ctx context.Context, userID uuid.UUID, subjectCode string, gradeLevel int) (*dto.ClassHeatmapResponse, error) {
	schoolID, err := s.repo.TeacherSchoolID(ctx, userID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, apperrors.NotFound("user not found")
		}
		return nil, apperrors.Internal(err)
	}

	// Keep the teacher's node + TEACHES_AT edge current in the graph.
	// Read path, so purely best-effort: the heatmap must not fail
	// because the mirror write did.
	_ = s.repo.SyncTeacherToNeo4j(ctx, userID, schoolID)

	classSize, err := s.repo.ClassSize(ctx, schoolID, gradeLevel)
	if err != nil {
		return nil, apperrors.Internal(err)
	}

	source := "neo4j"
	rows, err := s.repo.HeatmapNeo4j(ctx, schoolID, subjectCode, gradeLevel)
	if err != nil || len(rows) == 0 {
		source = "postgres"
		rows, err = s.repo.HeatmapPG(ctx, schoolID, subjectCode, gradeLevel)
		if err != nil {
			return nil, apperrors.Internal(err)
		}
	}

	resp := &dto.ClassHeatmapResponse{
		SubjectCode: subjectCode,
		GradeLevel:  gradeLevel,
		SchoolID:    schoolID.String(),
		ClassSize:   classSize,
		Source:      source,
		Topics:      make([]dto.HeatmapTopic, 0, len(rows)),
		Alerts:      []dto.HeatmapAlert{},
	}

	for _, row := range rows {
		pct := 0.0
		if classSize > 0 {
			pct = math.Round(float64(row.StudentsStruggling)/float64(classSize)*1000) / 10
		}
		topic := dto.HeatmapTopic{
			TopicID:            row.TopicID,
			Title:              row.Title,
			StudentsStruggling: row.StudentsStruggling,
			StrugglingPct:      pct,
			AvgSeverity:        math.Round(row.AvgSeverity*1000) / 1000,
		}

		if pct > alertThresholdPct {
			if alert := s.cohortAlert(ctx, schoolID, row, pct, gradeLevel, source); alert != nil {
				topic.Alert = alert
				resp.Alerts = append(resp.Alerts, *alert)
			}
		}
		resp.Topics = append(resp.Topics, topic)
	}
	return resp, nil
}

// cohortAlert traverses the prerequisite graph for a hot topic and, when
// a lower-grade prerequisite has cohort strugglers of its own, builds the
// teacher-facing warning. Uses the same store the heatmap came from so
// the numbers in one response are internally consistent.
func (s *Service) cohortAlert(ctx context.Context, schoolID uuid.UUID, row repository.HeatmapRow, pct float64, gradeLevel int, source string) *dto.HeatmapAlert {
	var (
		rc  *repository.RootCauseRow
		err error
	)
	if source == "neo4j" {
		rc, err = s.repo.CohortRootCauseNeo4j(ctx, schoolID, row.TopicID, gradeLevel)
		if err != nil || rc == nil {
			// Graph traversal failed or found nothing -- the PG
			// aggregation of 3A's per-student attributions can still
			// name a common cause.
			rc, err = s.repo.CohortRootCausePG(ctx, schoolID, row.TopicID, gradeLevel)
		}
	} else {
		rc, err = s.repo.CohortRootCausePG(ctx, schoolID, row.TopicID, gradeLevel)
	}
	if err != nil || rc == nil || rc.StudentsStruggling == 0 {
		return nil
	}

	return &dto.HeatmapAlert{
		TopicID:            row.TopicID,
		TopicTitle:         row.Title,
		StrugglingPct:      pct,
		RootCauseTopicID:   rc.TopicID,
		RootCauseTitle:     rc.Title,
		RootCauseGrade:     rc.GradeLevel,
		StudentsStruggling: rc.StudentsStruggling,
		Message: fmt.Sprintf(
			"⚠️ Warning: %.0f%% of your class is struggling with %s. Root cause analysis suggests Grade %d %s was not adequately mastered by this cohort.",
			pct, row.Title, rc.GradeLevel, rc.Title,
		),
	}
}
