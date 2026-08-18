// Package ai is a thin HTTP client for the Python ai-service, used by
// domains (career matching, gap analysis) that need model inference.
package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

type Client struct {
	baseURL string
	http    *http.Client
}

// NewClient accepts "host:port" or a full URL (AI_SERVICE_URL defaults to
// "ai-service:8000", scheme-less). Timeout must cover ai-service's own
// worst case, not just the fast local inference endpoints: the tutor's
// generate_with_fallback (app/utils/llm_provider.py) tries local Ollama
// first (OLLAMA_TIMEOUT_SECONDS=150s) and only falls back to Gemini
// (GEMINI_TIMEOUT_SECONDS=30s) if that fails/times out, so a single
// request can legitimately take up to ~180s under load before ai-service
// itself gives up. A 60s client timeout here used to be shorter than
// ai-service's own Ollama budget, so Go would abandon the request
// (surfacing as a bare "server disconnected" to the caller) while
// ai-service was still working -- caught via real local-LLM tutor calls
// under load, not a hypothetical. 210s gives headroom above the 150+30
// worst case.
func NewClient(baseURL string) *Client {
	if !strings.Contains(baseURL, "://") {
		baseURL = "http://" + baseURL
	}
	return &Client{baseURL: strings.TrimSuffix(baseURL, "/"), http: &http.Client{Timeout: 210 * time.Second}}
}

type CareerMatchRequest struct {
	StudentID string             `json:"student_id"`
	Grades    map[string]float64 `json:"grades"`
}

type CareerMatchResult struct {
	CareerPathID string  `json:"career_path_id"`
	Score        float64 `json:"score"`
}

type TutorAskRequest struct {
	StudentID string `json:"studentId"`
	Question  string `json:"question"`
	Language  string `json:"language"`
}

// TutorAskResponse mirrors the ai-service tutor's JSON (Capability 3C):
// the Gemini answer plus which curriculum topics and personal gap records
// were injected as context.
type TutorAskResponse struct {
	Answer        string          `json:"answer"`
	RelatedTopics json.RawMessage `json:"relatedTopics"`
	UsedGaps      json.RawMessage `json:"usedGaps"`
	Model         string          `json:"model"`
}

// TutorAsk calls POST /api/v1/tutor/ask on the ai-service.
func (c *Client) TutorAsk(ctx context.Context, req TutorAskRequest) (*TutorAskResponse, error) {
	var out TutorAskResponse
	if err := c.post(ctx, "/api/v1/tutor/ask", req, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// NationalInsights calls POST /api/v1/insights/national on the ai-service
// and returns the structured JSON response as a generic map. The caller
// (ministry/service/insights.go) is responsible for mapping the map into DTOs.
func (c *Client) NationalInsights(ctx context.Context, payload map[string]any) (map[string]any, error) {
	var out map[string]any
	if err := c.post(ctx, "/api/v1/insights/national", payload, &out); err != nil {
		return nil, err
	}
	return out, nil
}

// MatchCareers calls POST /api/v1/career/match on the ai-service.
func (c *Client) MatchCareers(ctx context.Context, req CareerMatchRequest) ([]CareerMatchResult, error) {
	var results []CareerMatchResult
	if err := c.post(ctx, "/api/v1/career/match", req, &results); err != nil {
		return nil, err
	}
	return results, nil
}

func (c *Client) post(ctx context.Context, path string, body, out any) error {
	payload, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+path, bytes.NewReader(payload))
	if err != nil {
		return fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("call ai-service %s: %w", path, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return fmt.Errorf("ai-service %s returned status %d", path, resp.StatusCode)
	}
	if out == nil {
		return nil
	}
	if err := json.NewDecoder(resp.Body).Decode(out); err != nil {
		return fmt.Errorf("decode ai-service response: %w", err)
	}
	return nil
}
