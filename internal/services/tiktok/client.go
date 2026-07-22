// Package tiktok provides a minimal client for fetching TikTok hashtag
// search results via Apify's clockworks/tiktok-scraper actor. It is the
// only place in the codebase that knows how to talk to Apify; the Trend
// Agent depends on it only through the Video DTO it returns.
package tiktok

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

const defaultBaseURL = "https://api.apify.com"

// runTimeoutSeconds is the Apify-side timeout passed to run-sync-get-dataset-items.
// clockworks/tiktok-scraper runs typically take 30-120s.
const runTimeoutSeconds = 120

// Client is a thin, authenticated HTTP client for running the
// clockworks/tiktok-scraper Apify actor synchronously.
type Client struct {
	apiToken   string
	actorID    string
	baseURL    string
	httpClient *http.Client
}

type Option func(*Client)

// WithBaseURL overrides the default API host, useful for tests.
func WithBaseURL(url string) Option {
	return func(c *Client) { c.baseURL = url }
}

// WithHTTPClient overrides the default http.Client, useful for tests.
func WithHTTPClient(hc *http.Client) Option {
	return func(c *Client) { c.httpClient = hc }
}

func NewClient(apiToken, actorID string, opts ...Option) *Client {
	c := &Client{
		apiToken: apiToken,
		actorID:  actorID,
		baseURL:  defaultBaseURL,
		httpClient: &http.Client{
			// Comfortably longer than the Apify-side run timeout so our
			// client never times out before Apify reports its own timeout.
			Timeout: (runTimeoutSeconds + 30) * time.Second,
		},
	}
	for _, opt := range opts {
		opt(c)
	}
	return c
}

type runSyncRequest struct {
	Hashtags       []string `json:"hashtags"`
	ResultsPerPage int      `json:"resultsPerPage"`
}

// FetchByHashtags runs the actor synchronously against the given hashtags
// and returns the matched videos. resultsPerPage must be set explicitly;
// the actor's own default is 1.
func (c *Client) FetchByHashtags(ctx context.Context, hashtags []string, resultsPerPage int) ([]Video, error) {
	payload, err := json.Marshal(runSyncRequest{Hashtags: hashtags, ResultsPerPage: resultsPerPage})
	if err != nil {
		return nil, fmt.Errorf("tiktok: encode request: %w", err)
	}

	path := fmt.Sprintf("/v2/acts/%s/run-sync-get-dataset-items?timeout=%d", c.actorID, runTimeoutSeconds)

	data, err := c.doWithRetries(ctx, http.MethodPost, path, payload)
	if err != nil {
		return nil, err
	}

	var items []apifyVideoItem
	if err := json.Unmarshal(data, &items); err != nil {
		return nil, fmt.Errorf("tiktok: decode response: %w", err)
	}

	videos := make([]Video, 0, len(items))
	for _, item := range items {
		videos = append(videos, item.toVideo())
	}
	return videos, nil
}

// doWithRetries sends an authenticated request, retrying transient failures
// (network errors, 429, 5xx) up to maxRetries times. 4xx responses other
// than 429 indicate a bad request and are not retried.
func (c *Client) doWithRetries(ctx context.Context, method, path string, body []byte) ([]byte, error) {
	const maxRetries = 2
	const retryDelay = 2 * time.Second

	var lastErr error
	for attempt := 0; attempt <= maxRetries; attempt++ {
		if attempt > 0 {
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(retryDelay):
			}
		}

		data, status, err := c.do(ctx, method, path, body)
		if err != nil {
			lastErr = err
			continue
		}
		if status == http.StatusTooManyRequests || status >= 500 {
			lastErr = fmt.Errorf("tiktok: unexpected status %d: %s", status, data)
			continue
		}
		// run-sync-get-dataset-items returns 201 (not 200) when it has to
		// start a fresh actor run rather than reuse a cached one.
		if status != http.StatusOK && status != http.StatusCreated {
			return nil, fmt.Errorf("tiktok: unexpected status %d: %s", status, data)
		}
		return data, nil
	}
	return nil, fmt.Errorf("tiktok: request failed after retries: %w", lastErr)
}

func (c *Client) do(ctx context.Context, method, path string, body []byte) ([]byte, int, error) {
	req, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, bytes.NewReader(body))
	if err != nil {
		return nil, 0, fmt.Errorf("tiktok: build request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+c.apiToken)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, 0, fmt.Errorf("tiktok: do request: %w", err)
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, 0, fmt.Errorf("tiktok: read response: %w", err)
	}
	return data, resp.StatusCode, nil
}
