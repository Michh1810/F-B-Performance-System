package menuidea

import (
	"context"
	"errors"
	"testing"
	"time"

	"fbperformance/internal/agents/trend"
	"fbperformance/internal/services/llm"
)

type fakeGenerator struct {
	response string
	err      error
	calls    int
	lastArgs struct {
		prompt string
		opts   llm.GenerateOptions
	}
}

func (f *fakeGenerator) Generate(ctx context.Context, model, prompt string, opts llm.GenerateOptions) (string, error) {
	f.calls++
	f.lastArgs.prompt = prompt
	f.lastArgs.opts = opts
	return f.response, f.err
}

type fakeEmbedder struct {
	embedding []float32
	err       error
	calls     int
}

func (f *fakeEmbedder) Embed(ctx context.Context, model, text string, opts llm.EmbedOptions) ([]float32, error) {
	f.calls++
	return f.embedding, f.err
}

// fakeSignalSearcher implements SignalSearcher with a single configurable
// result set, since GenerateIdeas now queries one relevance pool and
// classifies it in Go rather than querying two distance bands.
type fakeSignalSearcher struct {
	signals []RelevantSignal
	err     error
	calls   int
}

func (f *fakeSignalSearcher) SearchRelevant(ctx context.Context, embedding []float32, limit int, since time.Time) ([]RelevantSignal, error) {
	f.calls++
	if f.err != nil {
		return nil, f.err
	}
	return f.signals, nil
}

// tweakSignal returns a signal that talks about a related trend without
// naming the menu item — classified as tweak-candidate evidence.
func tweakSignal(views int64) []RelevantSignal {
	return []RelevantSignal{{
		Signal: trend.Signal{
			ExternalID: "1",
			Caption:    "birria ramen mashup",
			Hashtags:   []string{"foodtiktok", "birriaramen"},
			ViewCount:  views,
			LikeCount:  views / 10,
			URL:        "https://www.tiktok.com/@mockuser/video/1",
		},
		Distance: 0.25,
	}}
}

// promotionSignal returns a signal whose caption names the menu item
// directly — classified as promotion evidence.
func promotionSignal(itemName string, views int64) []RelevantSignal {
	return []RelevantSignal{{
		Signal: trend.Signal{
			ExternalID: "2",
			Caption:    "I'm shook!!! " + itemName + "!! #foodtiktok",
			Hashtags:   []string{"foodtiktok"},
			ViewCount:  views,
			LikeCount:  views / 10,
			URL:        "https://www.tiktok.com/@mockuser/video/2",
		},
		Distance: 0.32,
	}}
}

const validIdeaResponse = `{"ideas":[{"name":"Birria Ramen","description":"Ramen in birria consomme","suggested_category":"Noodles","rationale":"Trending fusion of two menu-adjacent items"}]}`

func TestGenerateIdeas_ZeroRelevantSignals(t *testing.T) {
	gen := &fakeGenerator{}
	embedder := &fakeEmbedder{embedding: []float32{0.1, 0.2}}
	searcher := &fakeSignalSearcher{}
	agent := NewAgent(gen, embedder, searcher, "test-model", "embed-model", 30)

	ideas, callLog, err := agent.GenerateIdeas(context.Background(), MenuItem{Name: "Birria Tacos", Category: "Tacos"})
	if err != nil {
		t.Fatalf("GenerateIdeas() error = %v, want nil", err)
	}
	if ideas != nil {
		t.Fatalf("expected nil ideas with zero relevant signals, got %v", ideas)
	}
	if gen.calls != 0 {
		t.Fatalf("expected Generator not called with zero relevant signals, got %d calls", gen.calls)
	}
	if callLog.AdjacentSignalCount != 0 || callLog.PromotionSignalCount != 0 {
		t.Fatalf("expected both counts 0, got adjacent=%d promotion=%d", callLog.AdjacentSignalCount, callLog.PromotionSignalCount)
	}
	if callLog.Error != "" {
		t.Fatalf("CallLog.Error = %q, want empty (zero signals isn't itself an error)", callLog.Error)
	}
}

func TestGenerateIdeas_EmbedFailureIsGraceful(t *testing.T) {
	gen := &fakeGenerator{}
	embedder := &fakeEmbedder{err: errors.New("embedding api down")}
	searcher := &fakeSignalSearcher{signals: tweakSignal(1000)}
	agent := NewAgent(gen, embedder, searcher, "test-model", "embed-model", 30)

	ideas, callLog, err := agent.GenerateIdeas(context.Background(), MenuItem{Name: "Birria Tacos", Category: "Tacos"})
	if err != nil {
		t.Fatalf("GenerateIdeas() error = %v, want nil (embed failure should degrade gracefully)", err)
	}
	if ideas != nil {
		t.Fatalf("expected nil ideas, got %v", ideas)
	}
	if searcher.calls != 0 {
		t.Fatalf("expected SearchRelevant not called after embed failure, got %d calls", searcher.calls)
	}
	if callLog.Error == "" {
		t.Fatalf("expected CallLog.Error to record the embed failure for debugging")
	}
}

func TestGenerateIdeas_SearchFailureIsGraceful(t *testing.T) {
	gen := &fakeGenerator{}
	embedder := &fakeEmbedder{embedding: []float32{0.1}}
	searcher := &fakeSignalSearcher{err: errors.New("db down")}
	agent := NewAgent(gen, embedder, searcher, "test-model", "embed-model", 30)

	ideas, callLog, err := agent.GenerateIdeas(context.Background(), MenuItem{Name: "Birria Tacos", Category: "Tacos"})
	if err != nil {
		t.Fatalf("GenerateIdeas() error = %v, want nil (search failure should degrade gracefully)", err)
	}
	if ideas != nil {
		t.Fatalf("expected nil ideas, got %v", ideas)
	}
	if gen.calls != 0 {
		t.Fatalf("expected Generator not called after search failure, got %d calls", gen.calls)
	}
	if callLog.Error == "" {
		t.Fatalf("expected CallLog.Error to record the search failure for debugging")
	}
}

func TestGenerateIdeas_LLMFailureHardFails(t *testing.T) {
	gen := &fakeGenerator{err: errors.New("generate down")}
	embedder := &fakeEmbedder{embedding: []float32{0.1}}
	searcher := &fakeSignalSearcher{signals: tweakSignal(1000)}
	agent := NewAgent(gen, embedder, searcher, "test-model", "embed-model", 30)

	ideas, callLog, err := agent.GenerateIdeas(context.Background(), MenuItem{Name: "Birria Tacos", Category: "Tacos"})
	if err == nil {
		t.Fatalf("expected GenerateIdeas() to hard-fail when the LLM call errors")
	}
	if ideas != nil {
		t.Fatalf("expected nil ideas on hard failure (no promotion evidence in this case), got %v", ideas)
	}
	if callLog.AdjacentSignalCount != 1 {
		t.Fatalf("CallLog.AdjacentSignalCount = %d, want 1 (still recorded even though the LLM call failed)", callLog.AdjacentSignalCount)
	}
	if callLog.PromptText == "" {
		t.Fatalf("expected CallLog.PromptText to be recorded even when the LLM call fails")
	}
	if callLog.Error == "" {
		t.Fatalf("expected CallLog.Error to record the generate failure")
	}
}

func TestGenerateIdeas_MalformedJSONIsError(t *testing.T) {
	gen := &fakeGenerator{response: "not json"}
	embedder := &fakeEmbedder{embedding: []float32{0.1}}
	searcher := &fakeSignalSearcher{signals: tweakSignal(1000)}
	agent := NewAgent(gen, embedder, searcher, "test-model", "embed-model", 30)

	_, callLog, err := agent.GenerateIdeas(context.Background(), MenuItem{Name: "Birria Tacos", Category: "Tacos"})
	if err == nil {
		t.Fatalf("expected GenerateIdeas() to error on malformed JSON, not panic or silently return empty")
	}
	if callLog.RawResponse != "not json" {
		t.Fatalf("CallLog.RawResponse = %q, want the raw malformed response preserved for debugging", callLog.RawResponse)
	}
}

func TestGenerateIdeas_HappyPath(t *testing.T) {
	gen := &fakeGenerator{response: validIdeaResponse}
	embedder := &fakeEmbedder{embedding: []float32{0.1}}
	searcher := &fakeSignalSearcher{signals: tweakSignal(1000)}
	agent := NewAgent(gen, embedder, searcher, "test-model", "embed-model", 30)

	ideas, callLog, err := agent.GenerateIdeas(context.Background(), MenuItem{Name: "Birria Tacos", Category: "Tacos"})
	if err != nil {
		t.Fatalf("GenerateIdeas() error = %v, want nil", err)
	}
	if len(ideas) != 1 {
		t.Fatalf("expected 1 idea, got %d", len(ideas))
	}
	got := ideas[0]
	if got.Kind != "tweak" {
		t.Fatalf("Kind = %q, want %q", got.Kind, "tweak")
	}
	if got.Name != "Birria Ramen" {
		t.Fatalf("Name = %q, want %q", got.Name, "Birria Ramen")
	}
	if got.SuggestedCategory != "Noodles" {
		t.Fatalf("SuggestedCategory = %q, want %q", got.SuggestedCategory, "Noodles")
	}
	if got.SourceSignalCount != 1 {
		t.Fatalf("SourceSignalCount = %d, want 1", got.SourceSignalCount)
	}
	if got.SourceTotalViews != 1000 {
		t.Fatalf("SourceTotalViews = %d, want 1000", got.SourceTotalViews)
	}
	if len(got.SourceHashtags) != 2 {
		t.Fatalf("SourceHashtags = %v, want 2 hashtags", got.SourceHashtags)
	}
	if len(got.SourceVideoURLs) != 1 || got.SourceVideoURLs[0] != "https://www.tiktok.com/@mockuser/video/1" {
		t.Fatalf("SourceVideoURLs = %v, want the one evidence clip URL", got.SourceVideoURLs)
	}
	if !gen.lastArgs.opts.JSONMode {
		t.Fatalf("expected the idea-generation call to use JSON mode")
	}

	if callLog.AdjacentSignalCount != 1 {
		t.Fatalf("CallLog.AdjacentSignalCount = %d, want 1", callLog.AdjacentSignalCount)
	}
	if callLog.PromotionSignalCount != 0 {
		t.Fatalf("CallLog.PromotionSignalCount = %d, want 0 (caption doesn't name the item)", callLog.PromotionSignalCount)
	}
	if len(callLog.PromptSignals) != 1 || callLog.PromptSignals[0].ExternalID != "1" {
		t.Fatalf("CallLog.PromptSignals = %+v, want the one signal shown to the LLM", callLog.PromptSignals)
	}
	if callLog.PromptSignals[0].URL != "https://www.tiktok.com/@mockuser/video/1" {
		t.Fatalf("CallLog.PromptSignals[0].URL = %q, want the signal's video URL", callLog.PromptSignals[0].URL)
	}
	if callLog.PromptText == "" {
		t.Fatalf("expected CallLog.PromptText to be recorded")
	}
	if callLog.RawResponse != validIdeaResponse {
		t.Fatalf("CallLog.RawResponse = %q, want the raw LLM response preserved", callLog.RawResponse)
	}
	if callLog.IdeaCount != 1 {
		t.Fatalf("CallLog.IdeaCount = %d, want 1", callLog.IdeaCount)
	}
	if callLog.Error != "" {
		t.Fatalf("CallLog.Error = %q, want empty on success", callLog.Error)
	}
}

func TestGenerateIdeas_EmptyIdeaListFromLLM(t *testing.T) {
	gen := &fakeGenerator{response: `{"ideas":[]}`}
	embedder := &fakeEmbedder{embedding: []float32{0.1}}
	searcher := &fakeSignalSearcher{signals: tweakSignal(1000)}
	agent := NewAgent(gen, embedder, searcher, "test-model", "embed-model", 30)

	ideas, callLog, err := agent.GenerateIdeas(context.Background(), MenuItem{Name: "Birria Tacos", Category: "Tacos"})
	if err != nil {
		t.Fatalf("GenerateIdeas() error = %v, want nil", err)
	}
	if len(ideas) != 0 {
		t.Fatalf("expected 0 ideas when the LLM finds nothing compelling, got %d", len(ideas))
	}
	if callLog.IdeaCount != 0 {
		t.Fatalf("CallLog.IdeaCount = %d, want 0", callLog.IdeaCount)
	}
}

func TestGenerateIdeas_PromotionOpportunity_NoLLMCallNeeded(t *testing.T) {
	gen := &fakeGenerator{}
	embedder := &fakeEmbedder{embedding: []float32{0.1}}
	searcher := &fakeSignalSearcher{signals: promotionSignal("Birria Tacos", 50000)}
	agent := NewAgent(gen, embedder, searcher, "test-model", "embed-model", 30)

	ideas, callLog, err := agent.GenerateIdeas(context.Background(), MenuItem{Name: "Birria Tacos", Category: "Tacos"})
	if err != nil {
		t.Fatalf("GenerateIdeas() error = %v, want nil", err)
	}
	if len(ideas) != 1 {
		t.Fatalf("expected 1 promotion idea, got %d: %+v", len(ideas), ideas)
	}
	got := ideas[0]
	if got.Kind != "promotion" {
		t.Fatalf("Kind = %q, want %q", got.Kind, "promotion")
	}
	if got.Name != "Birria Tacos" {
		t.Fatalf("Name = %q, want the existing item's own name %q", got.Name, "Birria Tacos")
	}
	if got.SuggestedCategory != "Tacos" {
		t.Fatalf("SuggestedCategory = %q, want the existing item's own category %q", got.SuggestedCategory, "Tacos")
	}
	if got.SourceTotalViews != 50000 {
		t.Fatalf("SourceTotalViews = %d, want 50000", got.SourceTotalViews)
	}
	if len(got.SourceVideoURLs) != 1 || got.SourceVideoURLs[0] != "https://www.tiktok.com/@mockuser/video/2" {
		t.Fatalf("SourceVideoURLs = %v, want the promotion signal's URL", got.SourceVideoURLs)
	}
	if gen.calls != 0 {
		t.Fatalf("expected no LLM call for a promotion idea (nothing to invent), got %d calls", gen.calls)
	}
	if callLog.PromotionSignalCount != 1 {
		t.Fatalf("CallLog.PromotionSignalCount = %d, want 1", callLog.PromotionSignalCount)
	}
	if callLog.AdjacentSignalCount != 0 {
		t.Fatalf("CallLog.AdjacentSignalCount = %d, want 0 (the only signal names the item)", callLog.AdjacentSignalCount)
	}
}

func TestGenerateIdeas_PromotionAndTweak_BothReturned(t *testing.T) {
	gen := &fakeGenerator{response: validIdeaResponse}
	embedder := &fakeEmbedder{embedding: []float32{0.1}}
	searcher := &fakeSignalSearcher{
		signals: append(promotionSignal("Birria Tacos", 50000), tweakSignal(1000)...),
	}
	agent := NewAgent(gen, embedder, searcher, "test-model", "embed-model", 30)

	ideas, _, err := agent.GenerateIdeas(context.Background(), MenuItem{Name: "Birria Tacos", Category: "Tacos"})
	if err != nil {
		t.Fatalf("GenerateIdeas() error = %v, want nil", err)
	}
	if len(ideas) != 2 {
		t.Fatalf("expected 1 promotion + 1 tweak idea, got %d: %+v", len(ideas), ideas)
	}
	if ideas[0].Kind != "promotion" || ideas[1].Kind != "tweak" {
		t.Fatalf("expected [promotion tweak] order, got [%s %s]", ideas[0].Kind, ideas[1].Kind)
	}
}

func TestMentionsItem(t *testing.T) {
	cases := []struct {
		caption  string
		itemName string
		want     bool
	}{
		{"I'm shook!!! Birria Tacos!! #birriatacos", "Birria Tacos", true},
		{"birria tacos with consomme, so good", "Birria Tacos", true}, // case-insensitive
		{"birria ramen fusion bowl", "Birria Tacos", false},
		{"", "Birria Tacos", false},
	}
	for _, c := range cases {
		if got := mentionsItem(c.caption, c.itemName); got != c.want {
			t.Errorf("mentionsItem(%q, %q) = %v, want %v", c.caption, c.itemName, got, c.want)
		}
	}
}
