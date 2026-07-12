// Package gemini provides a minimal authenticated HTTP client for the
// Google Gemini (Generative Language) API.
package gemini

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

const defaultBaseURL = "https://generativelanguage.googleapis.com"

// Client is a thin, authenticated HTTP client for the Gemini API.
type Client struct {
	apiKey     string
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

func NewClient(apiKey string, opts ...Option) *Client {
	c := &Client{
		apiKey:  apiKey,
		baseURL: defaultBaseURL,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
	for _, opt := range opts {
		opt(c)
	}
	return c
}

// do sends an authenticated request to the Gemini API. The API key is sent
// via the x-goog-api-key header, per Google's recommended auth scheme.
func (c *Client) do(ctx context.Context, method, path string, body io.Reader) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, body)
	if err != nil {
		return nil, fmt.Errorf("gemini: build request: %w", err)
	}
	req.Header.Set("x-goog-api-key", c.apiKey)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("gemini: do request: %w", err)
	}
	return resp, nil
}

type generateContentRequest struct {
	Contents []content `json:"contents"`
}

type content struct {
	Parts []part `json:"parts"`
}

type part struct {
	Text string `json:"text"`
}

type generateContentResponse struct {
	Candidates []struct {
		Content content `json:"content"`
	} `json:"candidates"`
}

// GenerateContent sends a single-turn prompt to the given model and returns
// the concatenated text of the first candidate's response.
func (c *Client) GenerateContent(ctx context.Context, model, prompt string) (string, error) {
	reqBody := generateContentRequest{
		Contents: []content{{Parts: []part{{Text: prompt}}}},
	}
	payload, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("gemini: encode request: %w", err)
	}

	path := fmt.Sprintf("/v1beta/models/%s:generateContent", model)
	resp, err := c.do(ctx, http.MethodPost, path, bytes.NewReader(payload))
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("gemini: read response: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("gemini: unexpected status %d: %s", resp.StatusCode, data)
	}

	var parsed generateContentResponse
	if err := json.Unmarshal(data, &parsed); err != nil {
		return "", fmt.Errorf("gemini: decode response: %w", err)
	}
	if len(parsed.Candidates) == 0 || len(parsed.Candidates[0].Content.Parts) == 0 {
		return "", fmt.Errorf("gemini: no candidates returned")
	}

	var text string
	for _, p := range parsed.Candidates[0].Content.Parts {
		text += p.Text
	}
	return text, nil
}

// ListModels calls GET /v1beta/models. It exists primarily to verify that a
// given API key authenticates correctly.
func (c *Client) ListModels(ctx context.Context) ([]byte, error) {
	resp, err := c.do(ctx, http.MethodGet, "/v1beta/models", nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("gemini: read response: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("gemini: unexpected status %d: %s", resp.StatusCode, data)
	}
	return data, nil
}
