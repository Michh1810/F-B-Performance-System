// Package financial implements the Financial Agent: it runs a demand
// forecast for a menu item (transaction history, a normalized decay
// baseline, and an optional AI-adjusted multiplier) and analyzes the result
// to produce a margin risk and demand outlook. It also exposes the
// forecast on its own via POST /api/forecast for callers that just want
// the raw numbers without the LLM writeup.
package financial

import (
	"context"
	"fmt"

	"fbperformance/internal/services/llm"
)

// defaultForecastHorizonDays is the forecast window the Financial Agent
// requests when analyzing a menu item as part of the recommendation
// pipeline. Callers of the standalone /api/forecast endpoint set their own
// horizon instead.
const defaultForecastHorizonDays = 7

type Agent struct {
	client     Generator
	model      string
	menuItems  MenuItemLookup
	forecaster Forecaster
}

func NewAgent(client Generator, model string, menuItems MenuItemLookup, forecaster Forecaster) *Agent {
	return &Agent{client: client, model: model, menuItems: menuItems, forecaster: forecaster}
}

// Analyze loads the menu item's pricing, runs a demand forecast against its
// transaction history, and asks the LLM to assess margin risk and demand
// outlook grounded in those numbers.
func (a *Agent) Analyze(ctx context.Context, in Input) (string, error) {
	item, err := a.menuItems.Get(ctx, in.MenuItemID)
	if err != nil {
		return "", fmt.Errorf("financial agent: load menu item: %w", err)
	}

	forecast, err := a.forecaster.ForecastMenuItems(ctx, []ForecastRequest{{
		ItemID:              in.MenuItemID.String(),
		ItemName:            in.ItemName,
		ForecastHorizonDays: defaultForecastHorizonDays,
		PriceCents:          item.PriceCents,
		EstimatedCOGSCents:  item.EstimatedCOGSCents,
	}})
	if err != nil {
		return "", fmt.Errorf("financial agent: forecast: %w", err)
	}

	text, err := a.client.Generate(ctx, a.model, buildPrompt(in, forecast.Forecasts[0]), llm.GenerateOptions{
		SystemPrompt: systemPrompt,
	})
	if err != nil {
		return "", fmt.Errorf("financial agent: %w", err)
	}
	return text, nil
}
