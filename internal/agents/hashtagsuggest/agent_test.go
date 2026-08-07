package hashtagsuggest

import (
	"context"
	"errors"
	"testing"

	"fbperformance/internal/services/llm"
)

type fakeGenerator struct {
	response string
	err      error
	lastArgs struct {
		prompt string
		opts   llm.GenerateOptions
	}
}

func (f *fakeGenerator) Generate(ctx context.Context, model, prompt string, opts llm.GenerateOptions) (string, error) {
	f.lastArgs.prompt = prompt
	f.lastArgs.opts = opts
	return f.response, f.err
}

func TestSuggest_HappyPath(t *testing.T) {
	gen := &fakeGenerator{response: `{"hashtags":["birriatacos","koreanfriedchicken","matchatok"]}`}
	agent := NewAgent(gen, "test-model")

	hashtags, err := agent.Suggest(context.Background(), Input{
		Description: "Korean-Mexican fusion fast casual",
		MenuItems:   []MenuItemRef{{Name: "Birria Tacos", Category: "Entree"}},
	})
	if err != nil {
		t.Fatalf("Suggest() error = %v, want nil", err)
	}
	if len(hashtags) != 3 {
		t.Fatalf("expected 3 hashtags, got %d: %v", len(hashtags), hashtags)
	}
	if !gen.lastArgs.opts.JSONMode {
		t.Fatalf("expected JSON mode to be requested")
	}
}

func TestSuggest_GeneratorError(t *testing.T) {
	gen := &fakeGenerator{err: errors.New("api down")}
	agent := NewAgent(gen, "test-model")

	_, err := agent.Suggest(context.Background(), Input{Description: "A coffee shop"})
	if err == nil {
		t.Fatalf("expected Suggest() to error when the generator fails")
	}
}

func TestSuggest_MalformedJSON(t *testing.T) {
	gen := &fakeGenerator{response: "not json"}
	agent := NewAgent(gen, "test-model")

	_, err := agent.Suggest(context.Background(), Input{Description: "A coffee shop"})
	if err == nil {
		t.Fatalf("expected Suggest() to error on malformed JSON")
	}
}

func TestSuggest_EmptyHashtagList(t *testing.T) {
	gen := &fakeGenerator{response: `{"hashtags":[]}`}
	agent := NewAgent(gen, "test-model")

	_, err := agent.Suggest(context.Background(), Input{Description: "A coffee shop"})
	if err == nil {
		t.Fatalf("expected Suggest() to error when the LLM returns no hashtags")
	}
}
