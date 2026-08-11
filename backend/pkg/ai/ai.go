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
// "ai-service:8000", scheme-less). Timeout covers the tutor's synchronous
// Gemini round-trip (the ai-service caps its own model call at 30s), not
// just the fast local inference endpoints.
func NewClient(baseURL string) *Client {
	if !strings.Contains(baseURL, "://") {
		baseURL = "http://" + baseURL
	}
	return &Client{baseURL: strings.TrimSuffix(baseURL, "/"), http: &http.Client{Timeout: 60 * time.Second}}
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

// MatchCareers calls POST /api/v1/career/match on the ai-service.
func (c *Client) MatchCareers(ctx context.Context, req CareerMatchRequest) ([]CareerMatchResult, error) {
	var results []CareerMatchResult
	if err := c.post(ctx, "/api/v1/career/match", req, &results); err != nil {
		return nil, err
	}
	return results, nil
}

type embedPassagesRequest struct {
	Texts []string `json:"texts"`
}

// EmbedResult carries back which embedding model actually produced the
// vectors (stored as embeddings.clo_embeddings.model_ver) alongside the
// vectors themselves, so a future model change is traceable per row
// instead of assumed from whatever EMBEDDING_MODEL happens to be
// configured today.
type EmbedResult struct {
	Embeddings [][]float32 `json:"embeddings"`
	Model      string      `json:"model"`
	Dim        int         `json:"dim"`
}

// EmbedPassages calls POST /api/v1/embeddings/passages on the ai-service,
// which runs each text through the local (non-cloud) embedding model --
// see ai-service/app/utils/embeddings.py. "Passages" (as opposed to
// "query") matches the e5-family embedding convention used by curriculum
// CLO descriptions, which are the documents being searched *against* when
// an exam question is matched to them (see exam_parser/vector_matcher.py
// on the ai-service side for the query-side counterpart).
func (c *Client) EmbedPassages(ctx context.Context, texts []string) (*EmbedResult, error) {
	var result EmbedResult
	if err := c.post(ctx, "/api/v1/embeddings/passages", embedPassagesRequest{Texts: texts}, &result); err != nil {
		return nil, err
	}
	return &result, nil
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
