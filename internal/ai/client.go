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

type AdjustBaselineInput struct {
	Candidate  CandidateContext
	Baseline   BaselineContext
	Comparable *ComparableContext
}

type CandidateContext struct {
	ItemID              string
	ItemName            string
	Category            string
	PriceCents          int64
	EstimatedCOGSCents  int64
	ForecastHorizonDays int
}

type BaselineContext struct {
	Model             string
	DailyUnits        float64
	HistoryWindowDays int
	Mode              string
	NonzeroSalesDays  int
	TotalUnits        int64
}

type ComparableContext struct {
	Count             int
	SyntheticBaseline float64
	Items             []ComparableItem
}

type ComparableItem struct {
	Name             string
	Category         string
	Baseline         float64
	SimilarityScore  float64
	Weight           float64
	NonzeroSalesDays int
	TotalUnits       int64
	Stage            string
}

func NewClientFromEnv() *Client {
	return &Client{APIKey: os.Getenv("AI_API_KEY"), Endpoint: os.Getenv("AI_ENDPOINT"), HTTPClient: http.DefaultClient}
}
func (c *Client) IsConfigured() bool { return c != nil && c.APIKey != "" && c.Endpoint != "" }

// AdjustBaseline calls a Gemini generateContent-compatible endpoint. AI_ENDPOINT
// must be the full endpoint URL; AI_API_KEY is sent as the x-goog-api-key header.
func (c *Client) AdjustBaseline(ctx context.Context, in AdjustBaselineInput) (float64, error) {
	if !c.IsConfigured() {
		return 1, fmt.Errorf("AI client is not configured")
	}
	prompt := buildAdjustmentPrompt(in)
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

func buildAdjustmentPrompt(in AdjustBaselineInput) string {
	var b strings.Builder
	fmt.Fprintf(&b, "You are adjusting an existing statistical demand baseline, not forecasting from scratch.\n")
	fmt.Fprintf(&b, "Use only the evidence below. Do not invent external facts.\n")
	fmt.Fprintf(&b, "Make only a bounded adjustment. If evidence is insufficient, return 1.0.\n\n")

	b.WriteString("Candidate:\n")
	fmt.Fprintf(&b, "- Item ID: %s\n", in.Candidate.ItemID)
	fmt.Fprintf(&b, "- Item name: %s\n", in.Candidate.ItemName)
	fmt.Fprintf(&b, "- Category: %s\n", in.Candidate.Category)
	fmt.Fprintf(&b, "- Price cents: %d\n", in.Candidate.PriceCents)
	fmt.Fprintf(&b, "- Estimated COGS cents: %d\n", in.Candidate.EstimatedCOGSCents)
	fmt.Fprintf(&b, "- Forecast horizon days: %d\n\n", in.Candidate.ForecastHorizonDays)

	b.WriteString("Statistical baseline evidence:\n")
	fmt.Fprintf(&b, "- Model: %s\n", in.Baseline.Model)
	fmt.Fprintf(&b, "- Baseline daily units: %.4f\n", in.Baseline.DailyUnits)
	fmt.Fprintf(&b, "- Forecast mode: %s\n", in.Baseline.Mode)
	fmt.Fprintf(&b, "- History window days: %d\n", in.Baseline.HistoryWindowDays)
	fmt.Fprintf(&b, "- Candidate nonzero sales days: %d\n", in.Baseline.NonzeroSalesDays)
	fmt.Fprintf(&b, "- Candidate total units: %d\n\n", in.Baseline.TotalUnits)

	if in.Comparable != nil {
		b.WriteString("Comparable cold-start evidence:\n")
		fmt.Fprintf(&b, "- Comparable count: %d\n", in.Comparable.Count)
		fmt.Fprintf(&b, "- Synthetic weighted baseline (daily units): %.4f\n", in.Comparable.SyntheticBaseline)
		for i, item := range in.Comparable.Items {
			fmt.Fprintf(&b, "- Comparable %d: name=%q, category=%q, baseline=%.4f, similarity_score=%.4f, weight=%.6f, nonzero_sales_days=%d, total_units=%d, stage=%s\n",
				i+1, item.Name, item.Category, item.Baseline, item.SimilarityScore, item.Weight, item.NonzeroSalesDays, item.TotalUnits, item.Stage)
		}
		b.WriteString("\n")
	}

	fmt.Fprintf(&b, "Output rules:\n")
	fmt.Fprintf(&b, "- Return only a decimal multiplier between %.1f and %.1f.\n", minAdjustmentMultiplier, maxAdjustmentMultiplier)
	b.WriteString("- Return only the number, with no JSON, markdown, explanation, or extra text.\n")
	b.WriteString("- Return 1.0 when no adjustment is justified.\n")

	return b.String()
}
