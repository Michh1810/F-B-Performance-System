package ai

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func sampleAdjustInput() AdjustBaselineInput {
	return AdjustBaselineInput{
		Candidate: CandidateContext{
			ItemID:              "11111111-1111-4111-8111-111111111111",
			ItemName:            "Salted Egg Coffee",
			Category:            "Beverage",
			PriceCents:          650,
			EstimatedCOGSCents:  200,
			ForecastHorizonDays: 7,
		},
		Baseline: BaselineContext{
			Model:             "normalized_exponential_decay_moving_average",
			DailyUnits:        12.3456,
			HistoryWindowDays: 30,
			Mode:              "own_history",
			NonzeroSalesDays:  5,
			TotalUnits:        84,
		},
		Comparable: &ComparableContext{
			Count:             2,
			SyntheticBaseline: 11.2,
			Items: []ComparableItem{
				{Name: "Iced Latte", Category: "Beverage", Baseline: 12.0, SimilarityScore: 0.9, Weight: 0.6, NonzeroSalesDays: 10, TotalUnits: 140, Stage: "same_category"},
				{Name: "Matcha Latte", Category: "Beverage", Baseline: 10.0, SimilarityScore: 0.8, Weight: 0.4, NonzeroSalesDays: 9, TotalUnits: 100, Stage: "same_category"},
			},
		},
	}
}

func TestAdjustBaseline_SuccessAndPromptContainsStructuredEvidence(t *testing.T) {
	input := sampleAdjustInput()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("x-goog-api-key"); got != "test-key" {
			t.Fatalf("x-goog-api-key = %q, want %q", got, "test-key")
		}

		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode request body: %v", err)
		}
		contents, ok := body["contents"].([]any)
		if !ok || len(contents) == 0 {
			t.Fatalf("contents missing or invalid: %#v", body["contents"])
		}
		firstContent, ok := contents[0].(map[string]any)
		if !ok {
			t.Fatalf("contents[0] invalid: %#v", contents[0])
		}
		parts, ok := firstContent["parts"].([]any)
		if !ok || len(parts) == 0 {
			t.Fatalf("parts missing or invalid: %#v", firstContent["parts"])
		}
		part0, ok := parts[0].(map[string]any)
		if !ok {
			t.Fatalf("parts[0] invalid: %#v", parts[0])
		}
		prompt, ok := part0["text"].(string)
		if !ok {
			t.Fatalf("prompt text missing: %#v", part0["text"])
		}

		mustContain := []string{
			"adjusting an existing statistical demand baseline",
			"Candidate:",
			"Item name: Salted Egg Coffee",
			"Category: Beverage",
			"Statistical baseline evidence:",
			"Forecast mode: own_history",
			"Comparable cold-start evidence:",
			"Return only a decimal multiplier between 0.5 and 1.5",
		}
		for _, want := range mustContain {
			if !strings.Contains(prompt, want) {
				t.Fatalf("prompt missing %q\nPrompt:\n%s", want, prompt)
			}
		}

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"candidates":[{"content":{"parts":[{"text":"1.08"}]}}]}`))
	}))
	defer server.Close()

	client := &Client{APIKey: "test-key", Endpoint: server.URL, HTTPClient: http.DefaultClient}
	multiplier, err := client.AdjustBaseline(context.Background(), input)
	if err != nil {
		t.Fatalf("AdjustBaseline() error = %v, want nil", err)
	}
	if multiplier != 1.08 {
		t.Fatalf("multiplier = %v, want 1.08", multiplier)
	}
}

func TestAdjustBaseline_RejectsOutOfRangeMultiplier(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"candidates":[{"content":{"parts":[{"text":"1.8"}]}}]}`))
	}))
	defer server.Close()

	client := &Client{APIKey: "test-key", Endpoint: server.URL, HTTPClient: http.DefaultClient}
	_, err := client.AdjustBaseline(context.Background(), sampleAdjustInput())
	if err == nil {
		t.Fatalf("expected out-of-range multiplier error")
	}
}

func TestAdjustBaseline_RejectsNonNumericMultiplier(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"candidates":[{"content":{"parts":[{"text":"about 10 percent"}]}}]}`))
	}))
	defer server.Close()

	client := &Client{APIKey: "test-key", Endpoint: server.URL, HTTPClient: http.DefaultClient}
	_, err := client.AdjustBaseline(context.Background(), sampleAdjustInput())
	if err == nil {
		t.Fatalf("expected parse error for non-numeric multiplier")
	}
}
