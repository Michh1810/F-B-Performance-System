// Package financial implements the Financial Agent: it analyzes POS/sales
// signals for a menu item and produces a margin risk and demand outlook.
package financial

import (
	"context"
	"fmt"

	"fbperformance/internal/services/llm"
)

type Agent struct {
	client Generator
	model  string
}

func NewAgent(client Generator, model string) *Agent {
	return &Agent{client: client, model: model}
}

// Analyze runs the Financial Agent over the given input and returns its
// free-text margin risk and demand outlook assessment.
func (a *Agent) Analyze(ctx context.Context, in Input) (string, error) {
	text, err := a.client.Generate(ctx, a.model, buildPrompt(in), llm.GenerateOptions{
		SystemPrompt: systemPrompt,
	})
	if err != nil {
		return "", fmt.Errorf("financial agent: %w", err)
	}
	return text, nil
}
