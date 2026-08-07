package financial

import (
	"context"

	"github.com/google/uuid"

	"fbperformance/internal/services/llm"
)

// Generator is the subset of the LLM client the Financial Agent depends on.
type Generator interface {
	Generate(ctx context.Context, model, prompt string, opts llm.GenerateOptions) (string, error)
}

// MenuItem is the pricing data the Financial Agent needs to price a
// forecast for a menu item.
type MenuItem struct {
	Name               string
	PriceCents         int64
	EstimatedCOGSCents int64
}

// MenuItemLookup is the subset of menu-item storage the Financial Agent
// depends on to load a menu item's pricing.
type MenuItemLookup interface {
	Get(ctx context.Context, id uuid.UUID) (MenuItem, error)
}

// Forecaster is the subset of demand-forecasting the Financial Agent
// depends on to ground its analysis in real sales numbers.
type Forecaster interface {
	ForecastMenuItems(ctx context.Context, requests []ForecastRequest) (ForecastResponse, error)
}

// Input is the menu item the Financial Agent analyzes.
type Input struct {
	MenuItemID uuid.UUID
	ItemName   string
}
