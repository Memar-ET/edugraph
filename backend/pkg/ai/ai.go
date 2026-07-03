// Package ai is a thin HTTP client for the Python ai-service, used by
// domains (career matching, gap analysis) that need model inference.
package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

type Client struct {
	baseURL string
	http    *http.Client
}

func NewClient(baseURL string) *Client {
	return &Client{baseURL: baseURL, http: &http.Client{Timeout: 15 * time.Second}}
}

type CareerMatchRequest struct {
	StudentID string             `json:"student_id"`
	Grades    map[string]float64 `json:"grades"`
}

type CareerMatchResult struct {
	CareerPathID string  `json:"career_path_id"`
	Score        float64 `json:"score"`
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
