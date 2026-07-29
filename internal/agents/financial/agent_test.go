package financial

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/google/uuid"

	"fbperformance/internal/services/llm"
)

type fakeGenerator struct {
	prompt string
	resp   string
	err    error
}

func (f *fakeGenerator) Generate(ctx context.Context, model, prompt string, opts llm.GenerateOptions) (string, error) {
	f.prompt = prompt
	return f.resp, f.err
}

type fakeMenuItemLookup struct {
	item MenuItem
	err  error
}

func (f *fakeMenuItemLookup) Get(ctx context.Context, id uuid.UUID) (MenuItem, error) {
	return f.item, f.err
}

type fakeForecaster struct {
	requests []ForecastRequest
	resp     ForecastResponse
	err      error
}

func (f *fakeForecaster) ForecastMenuItems(ctx context.Context, requests []ForecastRequest) (ForecastResponse, error) {
	f.requests = requests
	return f.resp, f.err
}

func TestAnalyze_UsesForecastToBuildPrompt(t *testing.T) {
	menuItemID := uuid.New()
	lookup := &fakeMenuItemLookup{item: MenuItem{Name: "Fusion Coffee", PriceCents: 650, EstimatedCOGSCents: 200}}
	forecaster := &fakeForecaster{resp: ForecastResponse{Forecasts: []ForecastResult{{
		ItemID: menuItemID.String(), ItemName: "Fusion Coffee", BaselineUnits: 12.5,
		ForecastedUnits: 87, ForecastedRevenueCents: 56550, ProjectedProfitCents: 39150,
		ForecastWindowDays: 7, Model: "normalized_exponential_decay_moving_average", AIAdjustmentStatus: "not_configured",
	}}}}
	gen := &fakeGenerator{resp: "margin risk is low"}
	agent := NewAgent(gen, "test-model", lookup, forecaster)

	text, err := agent.Analyze(context.Background(), Input{MenuItemID: menuItemID, ItemName: "Fusion Coffee"})
	if err != nil {
		t.Fatalf("Analyze() error = %v, want nil", err)
	}
	if text != "margin risk is low" {
		t.Fatalf("text = %q, want %q", text, "margin risk is low")
	}

	if len(forecaster.requests) != 1 {
		t.Fatalf("forecaster called with %d requests, want 1", len(forecaster.requests))
	}
	req := forecaster.requests[0]
	if req.ItemID != menuItemID.String() || req.PriceCents != 650 || req.EstimatedCOGSCents != 200 || req.ForecastHorizonDays != defaultForecastHorizonDays {
		t.Fatalf("unexpected forecast request: %+v", req)
	}

	if !strings.Contains(gen.prompt, "87") || !strings.Contains(gen.prompt, "Fusion Coffee") {
		t.Fatalf("prompt missing forecast numbers: %s", gen.prompt)
	}
}

func TestAnalyze_PropagatesMenuItemLookupError(t *testing.T) {
	lookup := &fakeMenuItemLookup{err: errors.New("not found")}
	agent := NewAgent(&fakeGenerator{}, "test-model", lookup, &fakeForecaster{})

	_, err := agent.Analyze(context.Background(), Input{MenuItemID: uuid.New(), ItemName: "Fusion Coffee"})
	if err == nil {
		t.Fatal("Analyze() error = nil, want error")
	}
}

func TestAnalyze_PropagatesForecastError(t *testing.T) {
	lookup := &fakeMenuItemLookup{item: MenuItem{Name: "Fusion Coffee"}}
	forecaster := &fakeForecaster{err: ErrNoHistoricalTransactions}
	agent := NewAgent(&fakeGenerator{}, "test-model", lookup, forecaster)

	_, err := agent.Analyze(context.Background(), Input{MenuItemID: uuid.New(), ItemName: "Fusion Coffee"})
	if !errors.Is(err, ErrNoHistoricalTransactions) {
		t.Fatalf("got %v, want wrapped ErrNoHistoricalTransactions", err)
	}
}
