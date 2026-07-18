package demand_forecast

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestHandlerAcceptsForecastContract(t *testing.T) {
	handler := NewHandler(NewService())
	body := `{"items":[{"item_id":"11111111-1111-4111-8111-111111111111","item_name":"Salted Egg Coffee","forecast_horizon_days":7,"price_cents":650,"estimated_cogs_cents":200,"historical_transactions":[{"timestamp":"2026-06-05T00:00:00Z","quantity":12}]}]}`
	request := httptest.NewRequest(http.MethodPost, "/api/forecast", strings.NewReader(body))
	response := httptest.NewRecorder()

	handler.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d: %s", http.StatusOK, response.Code, response.Body.String())
	}
}

func TestHandlerRejectsInvalidForecastContract(t *testing.T) {
	handler := NewHandler(NewService())
	body := `{"items":[{"item_id":"","item_name":"Salted Egg Coffee","forecast_horizon_days":0,"price_cents":-1,"estimated_cogs_cents":2}]}`
	request := httptest.NewRequest(http.MethodPost, "/api/forecast", strings.NewReader(body))
	response := httptest.NewRecorder()

	handler.ServeHTTP(response, request)

	if response.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d: %s", http.StatusBadRequest, response.Code, response.Body.String())
	}
	if !strings.Contains(response.Body.String(), "items[0].item_id is required") {
		t.Fatalf("expected field-level validation error, got %s", response.Body.String())
	}
}
