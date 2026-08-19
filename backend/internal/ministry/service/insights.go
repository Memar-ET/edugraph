package service

import (
	"context"
	"fmt"

	"github.com/edugraph-ai/edugraph/internal/ministry/dto"
	"github.com/edugraph-ai/edugraph/pkg/ai"
	apperrors "github.com/edugraph-ai/edugraph/pkg/errors"
)

// NationalInsights calls the ai-service to generate structured curriculum
// intelligence from the national overview stats.
func (s *Service) NationalInsights(ctx context.Context, overview dto.OverviewResponse) (dto.NationalInsightsResponse, error) {
	if s.ai == nil {
		return dto.NationalInsightsResponse{}, apperrors.Internal(fmt.Errorf("ai client not configured"))
	}

	payload := map[string]any{
		"total_regions":  overview.TotalRegions,
		"total_schools":  overview.TotalSchools,
		"total_students": overview.TotalStudents,
		"total_teachers": overview.TotalTeachers,
	}

	raw, err := s.ai.NationalInsights(ctx, payload)
	if err != nil {
		return dto.NationalInsightsResponse{}, apperrors.Internal(err)
	}

	resp := dto.NationalInsightsResponse{AIConfigured: true}

	if h, ok := raw["headline"].(string); ok {
		resp.Headline = h
	}
	if ts, ok := raw["trend_summary"].(string); ok {
		resp.TrendSummary = ts
	}

	if alerts, ok := raw["alerts"].([]any); ok {
		for _, a := range alerts {
			if m, ok := a.(map[string]any); ok {
				sev, _ := m["severity"].(string)
				msg, _ := m["message"].(string)
				if msg != "" {
					resp.Alerts = append(resp.Alerts, dto.InsightAlert{Severity: sev, Message: msg})
				}
			}
		}
	}
	if resp.Alerts == nil {
		resp.Alerts = []dto.InsightAlert{}
	}

	if recs, ok := raw["recommendations"].([]any); ok {
		for _, rec := range recs {
			if m, ok := rec.(map[string]any); ok {
				pri, _ := m["priority"].(string)
				txt, _ := m["text"].(string)
				if txt != "" {
					resp.Recommendations = append(resp.Recommendations, dto.InsightRecommendation{Priority: pri, Text: txt})
				}
			}
		}
	}
	if resp.Recommendations == nil {
		resp.Recommendations = []dto.InsightRecommendation{}
	}

	return resp, nil
}

// aiClientInterface is the subset of ai.Client methods this service uses,
// so we can dependency-inject without importing a concrete type.
type aiClientInterface interface {
	NationalInsights(ctx context.Context, payload map[string]any) (map[string]any, error)
}

// ensure *ai.Client satisfies the interface at compile time.
var _ aiClientInterface = (*ai.Client)(nil)
