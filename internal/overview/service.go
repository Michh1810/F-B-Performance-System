package overview

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"fbperformance/internal/services/llm"
)

type Service struct {
	repo      *Repository
	llmClient *llm.Client
	model     string
}

func NewService(repo *Repository, llmClient *llm.Client, model string) *Service {
	return &Service{repo: repo, llmClient: llmClient, model: model}
}

func calculateTrend(current, previous float64) (string, bool) {
	if previous == 0 {
		return "0%", true
	}
	change := ((current - previous) / previous) * 100
	sign := ""
	if change > 0 {
		sign = "+"
	}
	return fmt.Sprintf("%s%.1f%%", sign, change), change >= 0
}

func (s *Service) GetOverviewData(ctx context.Context, from, to time.Time) (OverviewResponse, error) {
	// Calculate previous period
	window := to.Sub(from)
	prevTo := from
	prevFrom := prevTo.Add(-window)

	current, err := s.repo.GetWeeklyMetrics(ctx, from, to)
	if err != nil {
		return OverviewResponse{}, fmt.Errorf("failed to get current metrics: %w", err)
	}
	
	previous, err := s.repo.GetWeeklyMetrics(ctx, prevFrom, prevTo)
	if err != nil {
		// Just log and continue, we can still show current metrics
		fmt.Printf("Warning: failed to get previous metrics: %v\n", err)
	}

	// Format WeeklyPulse
	pulse := WeeklyPulse{}

	revTrendStr, revTrendUp := calculateTrend(current.TotalRevenue, previous.TotalRevenue)
	pulse.Revenue = Metric{
		Value:   current.TotalRevenue,
		Label:   "Total Weekly Revenue",
		Trend:   revTrendStr,
		TrendUp: revTrendUp,
		Vs:      "previous period",
	}

	var currentAOV, prevAOV float64
	if current.TotalOrders > 0 {
		currentAOV = current.TotalRevenue / float64(current.TotalOrders)
	}
	if previous.TotalOrders > 0 {
		prevAOV = previous.TotalRevenue / float64(previous.TotalOrders)
	}
	aovTrendStr, aovTrendUp := calculateTrend(currentAOV, prevAOV)
	pulse.AOV = Metric{
		Value:   currentAOV,
		Label:   "Average Order Value",
		Trend:   aovTrendStr,
		TrendUp: aovTrendUp,
		Vs:      "previous period",
	}

	orderTrendStr, orderTrendUp := calculateTrend(float64(current.TotalOrders), float64(previous.TotalOrders))
	pulse.Orders = Metric{
		Value:   float64(current.TotalOrders),
		Label:   "Total Orders",
		Trend:   orderTrendStr,
		TrendUp: orderTrendUp,
		Vs:      "previous period",
	}

	// Sentiment is calculated simply by diffing the average rating
	sentimentTrend := current.AverageRating - previous.AverageRating
	sentimentTrendStr := fmt.Sprintf("%.2f", sentimentTrend)
	if sentimentTrend > 0 {
		sentimentTrendStr = "+" + sentimentTrendStr
	}
	pulse.Sentiment = Metric{
		Value:   current.AverageRating,
		Label:   "Overall Sentiment",
		Trend:   sentimentTrendStr,
		TrendUp: sentimentTrend >= 0,
		Vs:      fmt.Sprintf("previous period (%d new reviews)", current.TotalReviews),
	}

	// Evaluate Critical Alert
	var criticalAlert *CriticalAlert
	if current.TotalRevenue > 0 && previous.TotalRevenue > 0 {
		change := ((current.TotalRevenue - previous.TotalRevenue) / previous.TotalRevenue) * 100
		if change <= -15 {
			criticalAlert = &CriticalAlert{
				Active:  true,
				Type:    "destructive",
				Title:   "Action Required: Revenue Drop",
				Message: fmt.Sprintf("Total revenue dropped %.1f%% this period. View Performance Data for details.", -change),
			}
		} else if change >= 20 {
			criticalAlert = &CriticalAlert{
				Active:  true,
				Type:    "warning", // Positive alert, "warning" in UI just makes it yellow
				Title:   "Booming Sales: Revenue Spike",
				Message: fmt.Sprintf("Total revenue spiked %.1f%% this period! Ensure inventory and staffing can keep up.", change),
			}
		}
	} else if current.TotalRevenue > 0 && previous.TotalRevenue == 0 {
		criticalAlert = &CriticalAlert{
			Active:  true,
			Type:    "warning",
			Title:   "Baseline Data Collection",
			Message: "You have less than 30 days of historical data. We are tracking your current revenue and will begin generating critical alerts once enough comparative data is established.",
		}
	}

	// Generate AI Insights
	aiInsights, err := s.generateAIInsights(ctx, pulse)
	if err != nil {
		fmt.Printf("Warning: failed to generate AI insights: %v\n", err)
		aiInsights = []AIInsight{}
	}

	return OverviewResponse{
		CriticalAlert: criticalAlert,
		WeeklyPulse:   pulse,
		AIInsights:    aiInsights,
	}, nil
}

func (s *Service) generateAIInsights(ctx context.Context, pulse WeeklyPulse) ([]AIInsight, error) {
	prompt := fmt.Sprintf(`
You are an expert restaurant manager AI. Analyze the following weekly pulse data:
Revenue: $%.2f (Trend: %s)
AOV: $%.2f (Trend: %s)
Orders: %.0f (Trend: %s)
Sentiment: %.2f (Trend: %s)

Generate exactly 4 actionable insights for the restaurant owner.
Each insight must follow this exact JSON array structure:
[
  {
    "id": "marketing",
    "pillar": "Marketing & Promotions",
    "title": "Short title",
    "icon": "trendingUp", // one of: trendingUp, dollarSign, alertCircle, messageSquare
    "suggestion": "Actionable 1-2 sentence suggestion based on the data."
  }
]
Do not include any markdown formatting or extra text, just the raw JSON array.
`, pulse.Revenue.Value, pulse.Revenue.Trend, pulse.AOV.Value, pulse.AOV.Trend, pulse.Orders.Value, pulse.Orders.Trend, pulse.Sentiment.Value, pulse.Sentiment.Trend)

	opts := llm.GenerateOptions{
		SystemPrompt: "You are a helpful restaurant assistant that replies strictly in JSON.",
		JSONMode:     true,
	}
	
	resp, err := s.llmClient.Generate(ctx, s.model, prompt, opts)
	if err != nil {
		return nil, err
	}

	var insights []AIInsight
	if err := json.Unmarshal([]byte(resp), &insights); err != nil {
		return nil, fmt.Errorf("failed to parse LLM response: %w (raw: %s)", err, resp)
	}

	return insights, nil
}
