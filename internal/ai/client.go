package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"
)

const (
	minAdjustmentMultiplier = 0.5
	maxAdjustmentMultiplier = 1.5
)

type Client struct {
	APIKey, Endpoint string
	HTTPClient       *http.Client
}

func NewClientFromEnv() *Client {
	return &Client{APIKey: os.Getenv("AI_API_KEY"), Endpoint: os.Getenv("AI_ENDPOINT"), HTTPClient: http.DefaultClient}
}
func (c *Client) IsConfigured() bool { return c != nil && c.APIKey != "" && c.Endpoint != "" }

// AdjustBaseline calls a Gemini generateContent-compatible endpoint. AI_ENDPOINT
// must be the full endpoint URL; AI_API_KEY is sent as the x-goog-api-key header.
func (c *Client) AdjustBaseline(ctx context.Context, baseline float64, data map[string]any) (float64, error) {
	if !c.IsConfigured() {
		return 1, fmt.Errorf("AI client is not configured")
	}
	prompt := fmt.Sprintf("Return only a decimal multiplier between %.1f and %.1f for a daily sales baseline of %.4f. Context: %v. Use 1.0 when no adjustment is justified.", minAdjustmentMultiplier, maxAdjustmentMultiplier, baseline, data)
	body, err := json.Marshal(map[string]any{"contents": []any{map[string]any{"parts": []any{map[string]string{"text": prompt}}}}})
	if err != nil {
		return 1, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.Endpoint, bytes.NewReader(body))
	if err != nil {
		return 1, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-goog-api-key", c.APIKey)
	client := c.HTTPClient
	if client == nil {
		client = http.DefaultClient
	}
	resp, err := client.Do(req)
	if err != nil {
		return 1, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return 1, fmt.Errorf("AI endpoint returned %s", resp.Status)
	}
	var payload struct {
		Candidates []struct {
			Content struct {
				Parts []struct {
					Text string `json:"text"`
				} `json:"parts"`
			} `json:"content"`
		} `json:"candidates"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return 1, err
	}
	if len(payload.Candidates) == 0 || len(payload.Candidates[0].Content.Parts) == 0 {
		return 1, fmt.Errorf("AI response contained no multiplier")
	}
	multiplier, err := strconv.ParseFloat(strings.TrimSpace(payload.Candidates[0].Content.Parts[0].Text), 64)
	if err != nil || multiplier < minAdjustmentMultiplier || multiplier > maxAdjustmentMultiplier {
		return 1, fmt.Errorf("AI response multiplier must be between %.1f and %.1f", minAdjustmentMultiplier, maxAdjustmentMultiplier)
	}
	return multiplier, nil
}
