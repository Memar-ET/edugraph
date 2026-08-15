package dto

// InsightAlert is one urgent national-level flag.
type InsightAlert struct {
	Severity string `json:"severity"` // critical | warning | info
	Message  string `json:"message"`
}

// InsightRecommendation is one policy action item.
type InsightRecommendation struct {
	Priority string `json:"priority"` // high | medium | low
	Text     string `json:"text"`
}

// NationalInsightsResponse is the AI-generated curriculum insight for the Ministry dashboard.
type NationalInsightsResponse struct {
	Headline        string                  `json:"headline"`
	Alerts          []InsightAlert          `json:"alerts"`
	Recommendations []InsightRecommendation `json:"recommendations"`
	TrendSummary    string                  `json:"trend_summary"`
	AIConfigured    bool                    `json:"ai_configured"`
}
