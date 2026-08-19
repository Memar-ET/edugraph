package syncagent

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// CloudClient wraps all HTTP calls to the cloud EduGraph API.
// Auth uses two static headers (X-Device-Id / X-Device-Secret), matching
// the scheme defined in internal/sync/handler/device_auth.go — School Box
// devices are headless and have no user to issue JWTs.
type CloudClient struct {
	BaseURL    string
	DeviceID   string
	DeviceSecret string
	HTTPClient *http.Client
}

func NewCloudClient(baseURL, deviceID, deviceSecret string) *CloudClient {
	return &CloudClient{
		BaseURL:      baseURL,
		DeviceID:     deviceID,
		DeviceSecret: deviceSecret,
		HTTPClient:   &http.Client{Timeout: 30 * time.Second},
	}
}

func (c *CloudClient) newRequest(ctx context.Context, method, path string, body any) (*http.Request, error) {
	var buf io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("marshal: %w", err)
		}
		buf = bytes.NewReader(b)
	}
	req, err := http.NewRequestWithContext(ctx, method, c.BaseURL+path, buf)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Device-Id", c.DeviceID)
	req.Header.Set("X-Device-Secret", c.DeviceSecret)
	return req, nil
}

func (c *CloudClient) do(req *http.Request, out any) error {
	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return fmt.Errorf("http: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		b, _ := io.ReadAll(io.LimitReader(resp.Body, 2048))
		return fmt.Errorf("cloud returned %d: %s", resp.StatusCode, string(b))
	}
	if out != nil {
		return json.NewDecoder(resp.Body).Decode(out)
	}
	return nil
}

// PostSyncPush sends a batch of outbox rows to the cloud's /api/v1/sync/push.
func (c *CloudClient) PostSyncPush(ctx context.Context, schoolID string, rows []OutboxRow) (int, error) {
	items := make([]ChangeItem, 0, len(rows))
	for _, r := range rows {
		var payload map[string]any
		_ = json.Unmarshal(r.Payload, &payload)
		items = append(items, ChangeItem{
			EntityType: r.EntityType,
			EntityID:   r.EntityID,
			Operation:  r.Operation,
			Payload:    payload,
		})
	}
	req, err := c.newRequest(ctx, http.MethodPost, "/api/v1/sync/push", PushRequest{
		DeviceID: c.DeviceID,
		SchoolID: schoolID,
		Changes:  items,
	})
	if err != nil {
		return 0, err
	}
	var resp PushResponse
	if err := c.do(req, &resp); err != nil {
		return 0, err
	}
	return resp.Accepted, nil
}

// GetSyncPull fetches changes the cloud has for this school since `since`.
func (c *CloudClient) GetSyncPull(ctx context.Context, schoolID string, since time.Time) ([]CloudChange, error) {
	path := fmt.Sprintf("/api/v1/sync/pull?school_id=%s&since=%s",
		schoolID, since.UTC().Format(time.RFC3339))
	req, err := c.newRequest(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, err
	}
	var resp PullResponse
	if err := c.do(req, &resp); err != nil {
		return nil, err
	}
	return resp.Changes, nil
}
