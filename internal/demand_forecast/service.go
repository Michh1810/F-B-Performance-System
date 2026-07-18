package demand_forecast

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"time"

	"fbperformance/internal/ai"
)

const historyDays = 30

var ErrNoHistoricalTransactions = errors.New("no historical transactions available for forecast")

// Service forecasts daily sales with a normalized exponential-decay moving average.
type Service struct {
	db       *sql.DB
	aiClient *ai.Client
}

func NewService() *Service { return &Service{} }

func NewServiceWithDB(db *sql.DB, aiClient *ai.Client) *Service {
	return &Service{db: db, aiClient: aiClient}
}

func (s *Service) ForecastMenuItems(ctx context.Context, requests []ForecastRequest) (ForecastResponse, error) {
	results := make([]ForecastResult, 0, len(requests))
	for _, req := range requests {
		transactions := req.HistoricalTransactions
		var err error
		if len(transactions) == 0 && s.db != nil {
			transactions, err = s.loadDailyTransactionHistory(ctx, req.ItemID, historyDays)
			if err != nil {
				return ForecastResponse{}, err
			}
		}
		if len(transactions) == 0 {
			return ForecastResponse{}, fmt.Errorf("%w for item %s", ErrNoHistoricalTransactions, req.ItemID)
		}

		baseline := calculateNormalizedDecayAverage(transactions)
		multiplier := 1.0
		aiStatus := "not_configured"
		if s.aiClient != nil && s.aiClient.IsConfigured() {
			aiCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
			m, aiErr := s.aiClient.AdjustBaseline(aiCtx, baseline, map[string]any{
				"item_id": req.ItemID, "item_name": req.ItemName, "forecast_horizon_days": req.ForecastHorizonDays,
			})
			cancel()
			if aiErr == nil {
				multiplier = m
				aiStatus = "applied"
			} else {
				aiStatus = "failed"
			}
		}

		forecastedUnits := int(math.Round(baseline * float64(req.ForecastHorizonDays) * multiplier))
		forecastedRevenueCents := int64(forecastedUnits) * req.PriceCents
		projectedProfitCents := forecastedRevenueCents - int64(forecastedUnits)*req.EstimatedCOGSCents
		fr := ForecastResult{
			ItemID: req.ItemID, ItemName: req.ItemName, BaselineUnits: baseline,
			ForecastedUnits: forecastedUnits, ForecastedRevenueCents: forecastedRevenueCents,
			ProjectedProfitCents: projectedProfitCents, ForecastWindowDays: req.ForecastHorizonDays,
			Model:              "normalized_exponential_decay_moving_average",
			AIAdjustmentStatus: aiStatus,
		}

		if s.db != nil {
			persistCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
			err = s.persistForecast(persistCtx, fr, req.PriceCents, req.EstimatedCOGSCents, multiplier)
			cancel()
			if err != nil {
				return ForecastResponse{}, fmt.Errorf("persist forecast for item %s: %w", req.ItemID, err)
			}
		}
		results = append(results, fr)
	}
	return ForecastResponse{Forecasts: results}, nil
}

// loadDailyTransactionHistory returns one total quantity per day, newest first.
func (s *Service) loadDailyTransactionHistory(ctx context.Context, itemID string, days int) ([]TransactionHistory, error) {
	if days <= 0 {
		days = historyDays
	}
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	cutoff := time.Now().UTC().AddDate(0, 0, -days+1).Truncate(24 * time.Hour)
	rows, err := s.db.QueryContext(ctx, `
		WITH days AS (SELECT generate_series($2::date, CURRENT_DATE, INTERVAL '1 day')::timestamp AS sales_day)
		SELECT days.sales_day, COALESCE(SUM(t.quantity), 0)::bigint AS quantity
		FROM days LEFT JOIN transactions t ON t.menu_item_id = $1
			AND t.sold_at >= days.sales_day AND t.sold_at < days.sales_day + INTERVAL '1 day'
		GROUP BY days.sales_day ORDER BY days.sales_day DESC`, itemID, cutoff)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	transactions := make([]TransactionHistory, 0)
	for rows.Next() {
		var transaction TransactionHistory
		if err := rows.Scan(&transaction.Timestamp, &transaction.Quantity); err != nil {
			return nil, err
		}
		transactions = append(transactions, transaction)
	}
	return transactions, rows.Err()
}

func (s *Service) persistForecast(ctx context.Context, fr ForecastResult, priceCents, estimatedCOGSCents int64, multiplier float64) error {
	assumptions, err := json.Marshal(map[string]any{"ai_multiplier": multiplier, "ai_adjustment_status": fr.AIAdjustmentStatus, "baseline_unit": "daily_units", "history_days": historyDays})
	if err != nil {
		return err
	}
	_, err = s.db.ExecContext(ctx, `INSERT INTO forecasts (
		menu_item_id, model, baseline, forecasted_units, forecast_window_days, price,
		estimated_cogs, forecasted_revenue, projected_profit, assumptions, generated_at
	) VALUES ($1,$2,$3,$4,$5,$6::numeric / 100,$7::numeric / 100,$8::numeric / 100,$9::numeric / 100,$10,NOW())`, fr.ItemID, fr.Model,
		fr.BaselineUnits, fr.ForecastedUnits, fr.ForecastWindowDays, priceCents, estimatedCOGSCents,
		fr.ForecastedRevenueCents, fr.ProjectedProfitCents, assumptions)
	return err
}

// calculateNormalizedDecayAverage combines entries from the same date, sorts newest
// first, then gives each older day 70% of the weight of the prior day. Dividing by
// the sum of all weights keeps the result normalized regardless of history length.
func calculateNormalizedDecayAverage(transactions []TransactionHistory) float64 {
	daily := make(map[time.Time]int64)
	for _, transaction := range transactions {
		day := transaction.Timestamp.UTC().Truncate(24 * time.Hour)
		daily[day] += transaction.Quantity
	}
	var oldest, newest time.Time
	for day := range daily {
		if oldest.IsZero() || day.Before(oldest) {
			oldest = day
		}
		if newest.IsZero() || day.After(newest) {
			newest = day
		}
	}
	ordered := make([]TransactionHistory, 0, int(newest.Sub(oldest).Hours()/24)+1)
	for day := newest; !day.Before(oldest); day = day.AddDate(0, 0, -1) {
		ordered = append(ordered, TransactionHistory{Timestamp: day, Quantity: daily[day]})
	}
	var weightedSum, weightTotal float64
	for i, transaction := range ordered {
		weight := math.Pow(0.7, float64(i))
		weightedSum += float64(transaction.Quantity) * weight
		weightTotal += weight
	}
	return weightedSum / weightTotal
}
