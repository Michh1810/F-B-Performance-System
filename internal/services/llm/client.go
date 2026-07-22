// Package llm provides a minimal authenticated HTTP client for the
// Google Gemini (Generative Language) API. It is the only place in the
// codebase that knows how to talk to the LLM provider; agents depend on
// it only through the Generate method.
package llm

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
		return nil, fmt.Errorf("llm: build request: %w", err)
	}
	req.Header.Set("x-goog-api-key", c.apiKey)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("llm: do request: %w", err)
	}
	return resp, nil
}

type generateContentRequest struct {
	Contents          []content         `json:"contents"`
	SystemInstruction *content          `json:"systemInstruction,omitempty"`
	GenerationConfig  *generationConfig `json:"generationConfig,omitempty"`
}

type content struct {
	Parts []part `json:"parts"`
}

type part struct {
	Text string `json:"text"`
}

type generationConfig struct {
	ResponseMIMEType string `json:"responseMimeType,omitempty"`
}

type generateContentResponse struct {
	Candidates []struct {
		Content content `json:"content"`
	} `json:"candidates"`
}

// GenerateOptions customizes a single GenerateContent call.
type GenerateOptions struct {
	// SystemPrompt, if set, is sent as a systemInstruction to steer the
	// model's role/behavior separately from the user prompt.
	SystemPrompt string
	// JSONMode, if true, asks the model to return application/json so the
	// caller can unmarshal the response directly.
	JSONMode bool
}

// GenerateContent sends a single-turn prompt to the given model and returns
// the concatenated text of the first candidate's response.
func (c *Client) GenerateContent(ctx context.Context, model, prompt string) (string, error) {
	return c.Generate(ctx, model, prompt, GenerateOptions{})
}

// Generate sends a single-turn prompt to the given model, optionally with a
// system instruction and/or JSON-mode output, and returns the concatenated
// text of the first candidate's response.
func (c *Client) Generate(ctx context.Context, model, prompt string, opts GenerateOptions) (string, error) {
	reqBody := generateContentRequest{
		Contents: []content{{Parts: []part{{Text: prompt}}}},
	}
	if opts.SystemPrompt != "" {
		reqBody.SystemInstruction = &content{Parts: []part{{Text: opts.SystemPrompt}}}
	}
	if opts.JSONMode {
		reqBody.GenerationConfig = &generationConfig{ResponseMIMEType: "application/json"}
	}
	payload, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("llm: encode request: %w", err)
	}

	path := fmt.Sprintf("/v1beta/models/%s:generateContent", model)
	resp, err := c.do(ctx, http.MethodPost, path, bytes.NewReader(payload))
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("llm: read response: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("llm: unexpected status %d: %s", resp.StatusCode, data)
	}

	var parsed generateContentResponse
	if err := json.Unmarshal(data, &parsed); err != nil {
		return "", fmt.Errorf("llm: decode response: %w", err)
	}
	if len(parsed.Candidates) == 0 || len(parsed.Candidates[0].Content.Parts) == 0 {
		return "", fmt.Errorf("llm: no candidates returned")
	}

	var text string
	for _, p := range parsed.Candidates[0].Content.Parts {
		text += p.Text
	}
	return text, nil
}

// embedContentRequest's TaskType/OutputDimensionality are top-level fields,
// siblings of Content — NOT nested under an "embedContentConfig" object.
// Verified empirically against the live API: nesting them silently ignores
// OutputDimensionality and returns the model's native 3072-dim output.
type embedContentRequest struct {
	Content              content `json:"content"`
	TaskType             string  `json:"taskType,omitempty"`
	OutputDimensionality int     `json:"outputDimensionality,omitempty"`
}

type embedContentResponse struct {
	Embedding struct {
		Values []float32 `json:"values"`
	} `json:"embedding"`
}

// EmbedOptions customizes a single Embed call.
type EmbedOptions struct {
	// TaskType steers the embedding for its intended use, e.g.
	// "RETRIEVAL_DOCUMENT" for ingested content vs. "RETRIEVAL_QUERY" for a
	// search query — using the wrong one hurts retrieval quality.
	TaskType string
	// OutputDimensionality truncates the embedding to this many dimensions.
	// Must be consistent between ingestion-side and query-side calls, and
	// match the vector column width they're compared against.
	OutputDimensionality int
}

// Embed sends a single text to the given embedding model and returns its
// embedding vector.
func (c *Client) Embed(ctx context.Context, model, text string, opts EmbedOptions) ([]float32, error) {
	reqBody := embedContentRequest{
		Content:              content{Parts: []part{{Text: text}}},
		TaskType:             opts.TaskType,
		OutputDimensionality: opts.OutputDimensionality,
	}
	payload, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("llm: encode request: %w", err)
	}

	path := fmt.Sprintf("/v1beta/models/%s:embedContent", model)
	resp, err := c.do(ctx, http.MethodPost, path, bytes.NewReader(payload))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("llm: read response: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("llm: unexpected status %d: %s", resp.StatusCode, data)
	}

	var parsed embedContentResponse
	if err := json.Unmarshal(data, &parsed); err != nil {
		return nil, fmt.Errorf("llm: decode response: %w", err)
	}
	if len(parsed.Embedding.Values) == 0 {
		return nil, fmt.Errorf("llm: no embedding returned")
	}
	return parsed.Embedding.Values, nil
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
		return nil, fmt.Errorf("llm: read response: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("llm: unexpected status %d: %s", resp.StatusCode, data)
	}
	return data, nil
}
