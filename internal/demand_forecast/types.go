package demand_forecast

import "time"

// ForecastBatchRequest is the JSON contract accepted by POST /api/forecast.
// Historical transactions are optional: when omitted, the service loads the
// item's recent history from PostgreSQL.
type ForecastBatchRequest struct {
	Items []ForecastRequest `json:"items"`
}

type ForecastRequest struct {
	ItemID                 string               `json:"item_id"`
	ItemName               string               `json:"item_name"`
	HistoricalTransactions []TransactionHistory `json:"historical_transactions,omitempty"`
	ForecastHorizonDays    int                  `json:"forecast_horizon_days"`
	PriceCents             int64                `json:"price_cents"`
	EstimatedCOGSCents     int64                `json:"estimated_cogs_cents"`
}

type TransactionHistory struct {
	Timestamp time.Time `json:"timestamp"`
	Quantity  int64     `json:"quantity"`
}

type ForecastResponse struct {
	Forecasts []ForecastResult `json:"forecasts"`
}

type ForecastResult struct {
	ItemID                 string  `json:"item_id"`
	ItemName               string  `json:"item_name"`
	BaselineUnits          float64 `json:"baseline_units"`
	ForecastedUnits        int     `json:"forecasted_units"`
	ForecastedRevenueCents int64   `json:"forecasted_revenue_cents"`
	ProjectedProfitCents   int64   `json:"projected_profit_cents"`
	ForecastWindowDays     int     `json:"forecast_window_days"`
	Model                  string  `json:"model"`
	AIAdjustmentStatus     string  `json:"ai_adjustment_status"`
}
