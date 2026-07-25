// Package manager implements the Manager Agent: it synthesizes the Trend
// Agent and Financial Agent analyses for a menu item into a single
// structured LAUNCH/CUT/REPRICE decision.
package manager

import (
	"context"
	"encoding/json"
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

// Decide runs the Manager Agent over the given input and returns its
// structured decision.
func (a *Agent) Decide(ctx context.Context, in Input) (*Output, error) {
	text, err := a.client.Generate(ctx, a.model, buildPrompt(in), llm.GenerateOptions{
		SystemPrompt: systemPrompt,
		JSONMode:     true,
	})
	if err != nil {
		return nil, fmt.Errorf("manager agent: %w", err)
	}

	var out Output
	if err := json.Unmarshal([]byte(text), &out); err != nil {
		return nil, fmt.Errorf("manager agent: parse decision: %w (raw: %s)", err, text)
	}
	return &out, nil
}
