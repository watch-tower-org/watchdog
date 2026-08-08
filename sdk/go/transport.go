package wt

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

// ingestBasePath is the backend ingestion endpoint (relative to BaseURL).
const ingestBasePath = "/api/watchtower/v1/events"

// Result describes how the backend stored a single reported event. It mirrors
// backend/internal/model/IngestResult.
type Result struct {
	IssueID       int64  `json:"issue_id"`
	EventID       int64  `json:"event_id"`
	IsNewIssue    bool   `json:"is_new_issue"`
	WasRegression bool   `json:"was_regression"`
	Fingerprint   string `json:"fingerprint"`
}

type ingestResponse struct {
	Success bool `json:"success"`
	Data    struct {
		Received int      `json:"received"`
		Events   []Result `json:"events"`
	} `json:"data"`
}

// transport posts batches of events to the backend over HTTP.
type transport struct {
	baseURL string
	apiKey  string
	client  *http.Client
}

func newTransport(cfg Config) *transport {
	client := cfg.HTTPClient
	if client == nil {
		client = &http.Client{Timeout: cfg.HTTPTimeout}
	}
	return &transport{
		baseURL: strings.TrimRight(cfg.BaseURL, "/"),
		apiKey:  cfg.APIKey,
		client:  client,
	}
}

// send posts a batch of events. On failure (network error, non-2xx) it returns
// an error; the caller decides whether to drop or retry.
func (t *transport) send(ctx context.Context, events []event) ([]Result, error) {
	if len(events) == 0 {
		return nil, nil
	}
	body, err := json.Marshal(events)
	if err != nil {
		return nil, fmt.Errorf("watchtower: encode events: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, t.baseURL+ingestBasePath, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("watchtower: build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Api-Key", t.apiKey)

	resp, err := t.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("watchtower: send: %w", err)
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("watchtower: read response: %w", err)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("watchtower: server returned %d: %s", resp.StatusCode, strings.TrimSpace(string(data)))
	}

	var out ingestResponse
	if err := json.Unmarshal(data, &out); err != nil {
		return nil, fmt.Errorf("watchtower: invalid response: %w", err)
	}
	return out.Data.Events, nil
}
