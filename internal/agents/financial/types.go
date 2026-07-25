package financial

import (
	"context"

	"fbperformance/internal/services/llm"
)

// Generator is the subset of the LLM client the Financial Agent depends on.
type Generator interface {
	Generate(ctx context.Context, model, prompt string, opts llm.GenerateOptions) (string, error)
}

// Input is the data the Financial Agent analyzes for a single menu item.
type Input struct {
	ItemName      string
	FinancialData string
}
