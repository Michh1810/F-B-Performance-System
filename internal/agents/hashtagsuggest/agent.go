package hashtagsuggest

import (
	"context"
	"encoding/json"
	"fmt"

	"fbperformance/internal/services/llm"
)

// Agent suggests TikTok hashtags for cmd/trend-ingest to sweep, tailored to
// a specific restaurant's cuisine/style rather than the generic
// menu-item-agnostic default.
type Agent struct {
	client Generator
	model  string
}

func NewAgent(client Generator, model string) *Agent {
	return &Agent{client: client, model: model}
}

type hashtagResponse struct {
	Hashtags []string `json:"hashtags"`
}

// Suggest asks the LLM for a hashtag list grounded in in.Description and
// in.MenuItems. Unlike the batch agents (trend, menuidea), this is a
// synchronous, human-triggered call with no batch loop to continue past a
// failure — LLM errors, malformed JSON, and an empty hashtag list are all
// returned as hard failures for the caller (an HTTP handler) to surface
// directly, rather than degraded gracefully.
func (a *Agent) Suggest(ctx context.Context, in Input) ([]string, error) {
	text, err := a.client.Generate(ctx, a.model, buildPrompt(in), llm.GenerateOptions{
		SystemPrompt: systemPrompt,
		JSONMode:     true,
	})
	if err != nil {
		return nil, fmt.Errorf("hashtagsuggest agent: generate: %w", err)
	}

	var resp hashtagResponse
	if err := json.Unmarshal([]byte(text), &resp); err != nil {
		return nil, fmt.Errorf("hashtagsuggest agent: parse response: %w", err)
	}
	if len(resp.Hashtags) == 0 {
		return nil, fmt.Errorf("hashtagsuggest agent: LLM returned no hashtags")
	}
	return resp.Hashtags, nil
}
