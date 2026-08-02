package financial

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"sort"
	"time"

	"fbperformance/internal/ai"
)

const historyDays = 30

const (
	minNonzeroSalesDays     = 3
	minComparableItems      = 2
	maxComparableItems      = 5
	priceSimilarityWeight   = 0.60
	cogsSimilarityWeight    = 0.40
	categoryFallbackPenalty = 0.85
	maxRelativeDifference   = 2.0
	similarityScoreFloor    = 1e-9
	minimumPositiveBaseline = 0
)

var ErrNoHistoricalTransactions = errors.New("no historical transactions available for forecast")
var ErrNoUsableComparableItems = errors.New("no usable comparable menu items available for cold-start forecast")

type menuItemMeta struct {
	ID                 string
	Name               string
	Category           string
	PriceCents         int64
	EstimatedCOGSCents int64
	IsActive           bool
}

type comparableCandidate struct {
	menuItemMeta
	NonzeroSalesDays int
	TotalUnits       int64
	Baseline         float64
	Score            float64
	Weight           float64
	Stage            string
}

type historySummary struct {
	NonzeroSalesDays int
	TotalUnits       int64
}

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

		summary := summarizeHistory(transactions)
		baseline := calculateNormalizedDecayAverage(transactions)
		forecastMode := "own_history"
		comparablesAssumptions := []map[string]any{}

		if s.db != nil && !hasSufficientHistory(summary) {
			candidateMeta, metaErr := s.loadMenuItemMeta(ctx, req.ItemID)
			if metaErr != nil {
				if errors.Is(metaErr, sql.ErrNoRows) {
					return ForecastResponse{}, fmt.Errorf("candidate menu item %s not found: %w", req.ItemID, metaErr)
				}
				return ForecastResponse{}, metaErr
			}

			comparables, compErr := s.selectComparables(ctx, candidateMeta, req.PriceCents, req.EstimatedCOGSCents)
			if compErr != nil {
				if errors.Is(compErr, ErrNoUsableComparableItems) {
					return ForecastResponse{}, fmt.Errorf("%w for item %s", ErrNoUsableComparableItems, req.ItemID)
				}
				return ForecastResponse{}, compErr
			}

			baseline = weightedBaseline(comparables)
			forecastMode = "comparable_cold_start"
			comparablesAssumptions = comparableAssumptions(comparables)
		}
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

		assumptions := map[string]any{
			"ai_multiplier":          multiplier,
			"ai_adjustment_status":   fr.AIAdjustmentStatus,
			"baseline_unit":          "daily_units",
			"history_days":           historyDays,
			"forecast_mode":          forecastMode,
			"candidate_total_units":  summary.TotalUnits,
			"candidate_nonzero_days": summary.NonzeroSalesDays,
		}
		if forecastMode == "comparable_cold_start" {
			assumptions["min_nonzero_sales_days"] = minNonzeroSalesDays
			assumptions["min_comparables"] = minComparableItems
			assumptions["max_comparables"] = maxComparableItems
			assumptions["comparables"] = comparablesAssumptions
			assumptions["synthetic_baseline"] = baseline
		}

		if s.db != nil {
			persistCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
			err = s.persistForecast(persistCtx, fr, req.PriceCents, req.EstimatedCOGSCents, assumptions)
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

func (s *Service) persistForecast(ctx context.Context, fr ForecastResult, priceCents, estimatedCOGSCents int64, assumptionsData map[string]any) error {
	assumptions, err := json.Marshal(assumptionsData)
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

func summarizeHistory(transactions []TransactionHistory) historySummary {
	summary := historySummary{}
	for _, transaction := range transactions {
		if transaction.Quantity > 0 {
			summary.NonzeroSalesDays++
			summary.TotalUnits += transaction.Quantity
		}
	}
	return summary
}

func hasSufficientHistory(summary historySummary) bool {
	return summary.NonzeroSalesDays >= minNonzeroSalesDays && summary.TotalUnits > 0
}

func (s *Service) loadMenuItemMeta(ctx context.Context, itemID string) (menuItemMeta, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	var meta menuItemMeta
	var priceDollars, cogsDollars float64
	err := s.db.QueryRowContext(ctx, `
		SELECT id::text, name, category, current_price, cogs, is_active
		FROM menu_items
		WHERE id = $1`, itemID,
	).Scan(&meta.ID, &meta.Name, &meta.Category, &priceDollars, &cogsDollars, &meta.IsActive)
	if err != nil {
		return menuItemMeta{}, err
	}

	meta.PriceCents = int64(math.Round(priceDollars * 100))
	meta.EstimatedCOGSCents = int64(math.Round(cogsDollars * 100))
	return meta, nil
}

func (s *Service) listComparableCandidates(ctx context.Context, candidateID, candidateCategory string, sameCategoryOnly bool) ([]menuItemMeta, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	query := `
		SELECT id::text, name, category, current_price, cogs, is_active
		FROM menu_items
		WHERE is_active = true
		  AND id != $1`
	args := []any{candidateID}
	if sameCategoryOnly {
		query += ` AND category = $2`
		args = append(args, candidateCategory)
	}
	query += ` ORDER BY name, id`

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	candidates := make([]menuItemMeta, 0)
	for rows.Next() {
		var candidate menuItemMeta
		var priceDollars, cogsDollars float64
		if err := rows.Scan(&candidate.ID, &candidate.Name, &candidate.Category, &priceDollars, &cogsDollars, &candidate.IsActive); err != nil {
			return nil, err
		}
		candidate.PriceCents = int64(math.Round(priceDollars * 100))
		candidate.EstimatedCOGSCents = int64(math.Round(cogsDollars * 100))
		candidates = append(candidates, candidate)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}
	return candidates, nil
}

func (s *Service) selectComparables(ctx context.Context, candidate menuItemMeta, candidatePriceCents, candidateCOGSCents int64) ([]comparableCandidate, error) {
	sameCategoryCandidates, err := s.listComparableCandidates(ctx, candidate.ID, candidate.Category, true)
	if err != nil {
		return nil, err
	}

	stageA, err := s.buildUsableComparables(ctx, sameCategoryCandidates, candidate, candidatePriceCents, candidateCOGSCents, true)
	if err != nil {
		return nil, err
	}
	if len(stageA) >= minComparableItems {
		selected := stageA
		if len(selected) > maxComparableItems {
			selected = selected[:maxComparableItems]
		}
		normalizeWeights(selected)
		return selected, nil
	}

	allCandidates, err := s.listComparableCandidates(ctx, candidate.ID, candidate.Category, false)
	if err != nil {
		return nil, err
	}

	stageB, err := s.buildUsableComparables(ctx, allCandidates, candidate, candidatePriceCents, candidateCOGSCents, false)
	if err != nil {
		return nil, err
	}
	if len(stageB) == 0 {
		return nil, ErrNoUsableComparableItems
	}

	selected := stageB
	if len(selected) > maxComparableItems {
		selected = selected[:maxComparableItems]
	}
	normalizeWeights(selected)
	return selected, nil
}

func (s *Service) buildUsableComparables(ctx context.Context, candidates []menuItemMeta, candidate menuItemMeta, candidatePriceCents, candidateCOGSCents int64, sameCategoryStage bool) ([]comparableCandidate, error) {
	usable := make([]comparableCandidate, 0, len(candidates))
	for _, candidateItem := range candidates {
		history, err := s.loadDailyTransactionHistory(ctx, candidateItem.ID, historyDays)
		if err != nil {
			continue
		}
		historyStats := summarizeHistory(history)
		if !hasSufficientHistory(historyStats) {
			continue
		}
		baseline := calculateNormalizedDecayAverage(history)
		if baseline <= minimumPositiveBaseline {
			continue
		}

		score := combinedSimilarityScore(candidatePriceCents, candidateCOGSCents, candidate, candidateItem, sameCategoryStage)
		stage := "same_category"
		if !sameCategoryStage {
			stage = "cross_category_fallback"
		}
		usable = append(usable, comparableCandidate{
			menuItemMeta:     candidateItem,
			NonzeroSalesDays: historyStats.NonzeroSalesDays,
			TotalUnits:       historyStats.TotalUnits,
			Baseline:         baseline,
			Score:            score,
			Stage:            stage,
		})
	}

	sortComparables(usable)
	return usable, nil
}

func sortComparables(candidates []comparableCandidate) {
	sort.Slice(candidates, func(i, j int) bool {
		left := candidates[i]
		right := candidates[j]
		if left.Score != right.Score {
			return left.Score > right.Score
		}
		if left.NonzeroSalesDays != right.NonzeroSalesDays {
			return left.NonzeroSalesDays > right.NonzeroSalesDays
		}
		if left.Baseline != right.Baseline {
			return left.Baseline > right.Baseline
		}
		return left.ID < right.ID
	})
}

func clampRelativeDifference(value float64) float64 {
	if value < 0 {
		return 0
	}
	if value > maxRelativeDifference {
		return maxRelativeDifference
	}
	return value
}

func relativeSimilarity(candidateValue, comparableValue int64) float64 {
	denominator := math.Max(float64(candidateValue), 1)
	diff := math.Abs(float64(comparableValue-candidateValue)) / denominator
	diff = clampRelativeDifference(diff)
	return 1.0 / (1.0 + diff)
}

func combinedSimilarityScore(candidatePriceCents, candidateCOGSCents int64, candidate, comparable menuItemMeta, sameCategoryStage bool) float64 {
	priceScore := relativeSimilarity(candidatePriceCents, comparable.PriceCents)
	cogsScore := relativeSimilarity(candidateCOGSCents, comparable.EstimatedCOGSCents)
	base := priceSimilarityWeight*priceScore + cogsSimilarityWeight*cogsScore
	if sameCategoryStage {
		return base
	}
	if candidate.Category == comparable.Category {
		return base
	}
	return base * categoryFallbackPenalty
}

func normalizeWeights(candidates []comparableCandidate) {
	total := 0.0
	for i := range candidates {
		raw := math.Max(candidates[i].Score, similarityScoreFloor)
		total += raw
		candidates[i].Weight = raw
	}
	if total <= 0 {
		return
	}
	for i := range candidates {
		candidates[i].Weight = candidates[i].Weight / total
	}
}

func weightedBaseline(candidates []comparableCandidate) float64 {
	baseline := 0.0
	for _, candidate := range candidates {
		baseline += candidate.Baseline * candidate.Weight
	}
	return baseline
}

func comparableAssumptions(candidates []comparableCandidate) []map[string]any {
	assumptions := make([]map[string]any, 0, len(candidates))
	for _, candidate := range candidates {
		assumptions = append(assumptions, map[string]any{
			"menu_item_id":       candidate.ID,
			"name":               candidate.Name,
			"category":           candidate.Category,
			"stage":              candidate.Stage,
			"baseline":           candidate.Baseline,
			"similarity_score":   candidate.Score,
			"weight":             candidate.Weight,
			"nonzero_sales_days": candidate.NonzeroSalesDays,
			"total_units":        candidate.TotalUnits,
		})
	}
	return assumptions
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
