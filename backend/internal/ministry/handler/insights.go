package handler

import (
	"net/http"

	"github.com/edugraph-ai/edugraph/internal/ministry/dto"
	"github.com/edugraph-ai/edugraph/pkg/middleware"
)

var insightsFallback = dto.NationalInsightsResponse{
	Headline:        "National curriculum data available — AI insights require LLM configuration.",
	Alerts:          []dto.InsightAlert{},
	Recommendations: []dto.InsightRecommendation{},
	TrendSummary:    "",
	AIConfigured:    false,
}

// CurriculumInsights handles POST /api/v1/ministry/curriculum-insights.
// Gathers the national overview, calls the ai-service for structured insights,
// and returns them to the Ministry dashboard. Degrades gracefully to a
// deterministic fallback when the AI service is unavailable.
func (h *Handler) CurriculumInsights(w http.ResponseWriter, r *http.Request) {
	overview, err := h.svc.Overview(r.Context())
	if err != nil {
		middleware.WriteError(w, err)
		return
	}

	insights, err := h.svc.NationalInsights(r.Context(), overview)
	if err != nil {
		// AI service is down or unconfigured — return the fallback so the
		// Ministry dashboard renders rather than showing a 500.
		middleware.WriteJSON(w, http.StatusOK, insightsFallback)
		return
	}
	middleware.WriteJSON(w, http.StatusOK, insights)
}
