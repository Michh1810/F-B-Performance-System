package financial

import (
	"context"
	"errors"
	"math"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"fbperformance/internal/ai"
)

func TestForecastMenuItemsCalculatesDailyBaselineAndHorizon(t *testing.T) {
	request := ForecastRequest{ItemID: "11111111-1111-4111-8111-111111111111", ItemName: "Salted Egg Coffee", ForecastHorizonDays: 3, PriceCents: 650, EstimatedCOGSCents: 200, HistoricalTransactions: []TransactionHistory{
		{Timestamp: time.Date(2026, 6, 3, 15, 0, 0, 0, time.UTC), Quantity: 10}, {Timestamp: time.Date(2026, 6, 2, 0, 0, 0, 0, time.UTC), Quantity: 5}, {Timestamp: time.Date(2026, 6, 3, 9, 0, 0, 0, time.UTC), Quantity: 2}, {Timestamp: time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC), Quantity: 1},
	}}
	response, err := NewService().ForecastMenuItems(context.Background(), []ForecastRequest{request})
	if err != nil {
		t.Fatal(err)
	}
	expected := (12 + 5*.7 + 1*.49) / (1 + .7 + .49)
	if math.Abs(response.Forecasts[0].BaselineUnits-expected) > .0001 {
		t.Fatalf("baseline = %v, want %v", response.Forecasts[0].BaselineUnits, expected)
	}
	if response.Forecasts[0].ForecastedUnits != int(math.Round(expected*3)) {
		t.Fatal("forecast horizon was not applied")
	}
}

func TestForecastMenuItemsRejectsNoHistory(t *testing.T) {
	_, err := NewService().ForecastMenuItems(context.Background(), []ForecastRequest{{ItemID: "11111111-1111-4111-8111-111111111111"}})
	if !errors.Is(err, ErrNoHistoricalTransactions) {
		t.Fatalf("got %v", err)
	}
}

func TestNormalizedDecayAverageIncludesZeroSalesDays(t *testing.T) {
	baseline := calculateNormalizedDecayAverage([]TransactionHistory{
		{Timestamp: time.Date(2026, 6, 3, 0, 0, 0, 0, time.UTC), Quantity: 10},
		{Timestamp: time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC), Quantity: 10},
	})
	expected := (10 + 0*.7 + 10*.49) / (1 + .7 + .49)
	if math.Abs(baseline-expected) > .0001 {
		t.Fatalf("baseline = %v, want %v", baseline, expected)
	}
}

func TestForecastMenuItems_AIAppliedAdjustsUnits(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"candidates":[{"content":{"parts":[{"text":"1.1"}]}}]}`))
	}))
	defer server.Close()

	service := NewServiceWithDB(nil, &ai.Client{APIKey: "test-key", Endpoint: server.URL, HTTPClient: http.DefaultClient})
	request := ForecastRequest{
		ItemID:              "11111111-1111-4111-8111-111111111111",
		ItemName:            "Salted Egg Coffee",
		ForecastHorizonDays: 3,
		PriceCents:          650,
		EstimatedCOGSCents:  200,
		HistoricalTransactions: []TransactionHistory{
			{Timestamp: time.Date(2026, 6, 3, 15, 0, 0, 0, time.UTC), Quantity: 10},
			{Timestamp: time.Date(2026, 6, 2, 0, 0, 0, 0, time.UTC), Quantity: 5},
			{Timestamp: time.Date(2026, 6, 3, 9, 0, 0, 0, time.UTC), Quantity: 2},
			{Timestamp: time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC), Quantity: 1},
		},
	}

	response, err := service.ForecastMenuItems(context.Background(), []ForecastRequest{request})
	if err != nil {
		t.Fatal(err)
	}

	baseline := (12 + 5*.7 + 1*.49) / (1 + .7 + .49)
	wantUnits := int(math.Round(baseline * 3 * 1.1))
	if response.Forecasts[0].ForecastedUnits != wantUnits {
		t.Fatalf("forecasted units = %d, want %d", response.Forecasts[0].ForecastedUnits, wantUnits)
	}
	if response.Forecasts[0].AIAdjustmentStatus != "applied" {
		t.Fatalf("ai adjustment status = %q, want %q", response.Forecasts[0].AIAdjustmentStatus, "applied")
	}
}

func TestForecastMenuItems_AIInvalidMultiplierFallsBackSafely(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"candidates":[{"content":{"parts":[{"text":"not-a-number"}]}}]}`))
	}))
	defer server.Close()

	service := NewServiceWithDB(nil, &ai.Client{APIKey: "test-key", Endpoint: server.URL, HTTPClient: http.DefaultClient})
	request := ForecastRequest{
		ItemID:              "11111111-1111-4111-8111-111111111111",
		ItemName:            "Salted Egg Coffee",
		ForecastHorizonDays: 3,
		PriceCents:          650,
		EstimatedCOGSCents:  200,
		HistoricalTransactions: []TransactionHistory{
			{Timestamp: time.Date(2026, 6, 3, 15, 0, 0, 0, time.UTC), Quantity: 10},
			{Timestamp: time.Date(2026, 6, 2, 0, 0, 0, 0, time.UTC), Quantity: 5},
			{Timestamp: time.Date(2026, 6, 3, 9, 0, 0, 0, time.UTC), Quantity: 2},
			{Timestamp: time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC), Quantity: 1},
		},
	}

	response, err := service.ForecastMenuItems(context.Background(), []ForecastRequest{request})
	if err != nil {
		t.Fatal(err)
	}

	baseline := (12 + 5*.7 + 1*.49) / (1 + .7 + .49)
	wantUnits := int(math.Round(baseline * 3))
	if response.Forecasts[0].ForecastedUnits != wantUnits {
		t.Fatalf("forecasted units = %d, want %d", response.Forecasts[0].ForecastedUnits, wantUnits)
	}
	if response.Forecasts[0].AIAdjustmentStatus != "failed" {
		t.Fatalf("ai adjustment status = %q, want %q", response.Forecasts[0].AIAdjustmentStatus, "failed")
	}
}
