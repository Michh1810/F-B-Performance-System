package menuidea

import (
	"context"
	"time"

	"github.com/google/uuid"

	"fbperformance/internal/agents/trend"
	"fbperformance/internal/services/llm"
)

// EmbeddingDimensions matches trend.EmbeddingDimensions: this agent embeds
// menu items into, and searches, the same trend_signals corpus and model.
const EmbeddingDimensions = trend.EmbeddingDimensions

// Generator is the subset of the LLM client this agent depends on to
// synthesize idea candidates from adjacent trend evidence.
type Generator interface {
	Generate(ctx context.Context, model, prompt string, opts llm.GenerateOptions) (string, error)
}

// Embedder is the subset of the LLM client this agent depends on to embed a
// menu item as a query concept for adjacency search.
type Embedder interface {
	Embed(ctx context.Context, model, text string, opts llm.EmbedOptions) ([]float32, error)
}

// MenuItem is the minimal shape this agent needs to build a query embedding
// for an existing menu item — kept local rather than reusing
// financial.MenuItem (which carries pricing, not category) or
// performance_analytics' MenuItem (an unrelated read model).
type MenuItem struct {
	ID       uuid.UUID
	Name     string
	Category string
}

// RelevantSignal is a trend signal found meaningfully related to a menu
// item's embedding, plus the distance that placed it there (see
// SignalSearcher). Whether it describes the item itself ("promotion"
// evidence) or something distinct ("tweak" evidence) is decided by text —
// see Agent.GenerateIdeas — not by distance: real embeddings don't
// reliably separate the two by distance alone (see relevanceCutoff's doc
// comment in internal/store/trendsignals.go for the data that showed this).
type RelevantSignal struct {
	trend.Signal
	Distance float64
}

// SignalSearcher is the subset of relevance search this agent depends on.
type SignalSearcher interface {
	// SearchRelevant returns signals whose embedding is meaningfully
	// related to embedding, most-similar first, filtered to those posted
	// at or after since. This is the single evidence pool for one menu
	// item — GenerateIdeas splits it into promotion vs tweak evidence by
	// caption text, not by a second distance band.
	SearchRelevant(ctx context.Context, embedding []float32, limit int, since time.Time) ([]RelevantSignal, error)
}

// IdeaCandidate is one proposed idea grounded in trend evidence — either an
// LLM-proposed new item ("tweak", adjacent-band evidence) or a
// deterministically-synthesized call to feature an existing item
// ("promotion", near-duplicate-band evidence). See Kind.
type IdeaCandidate struct {
	// Kind is "tweak" (a new item, distinct from the existing one, proposed
	// by the LLM from adjacent trend evidence) or "promotion" (the existing
	// item itself, surfaced because near-identical content is trending —
	// synthesized directly from the evidence, no LLM call, since there's
	// nothing to invent).
	Kind string `json:"kind"`

	Name              string `json:"name"`
	Description       string `json:"description"`
	SuggestedCategory string `json:"suggested_category"`
	Rationale         string `json:"rationale"`

	SourceHashtags    []string `json:"source_hashtags"`
	SourceSignalCount int      `json:"source_signal_count"`
	SourceTotalViews  int64    `json:"source_total_views"`
	// SourceVideoURLs links back to the actual TikTok videos that fed this
	// idea, deduped in the same first-seen order as SourceHashtags — the
	// "evidence clips" a human reviewer can click through to, rather than
	// judging the idea on caption text alone. Empty for signals ingested
	// before video_url existed.
	SourceVideoURLs []string `json:"source_video_urls"`
}

// StoredIdea wraps an IdeaCandidate with the persistence fields needed for
// the review workflow (menu_ideas table).
type StoredIdea struct {
	ID                     uuid.UUID  `json:"id"`
	BatchRunID             uuid.UUID  `json:"batch_run_id"`
	InspiredByMenuItemID   uuid.UUID  `json:"inspired_by_menu_item_id"`
	InspiredByMenuItemName string     `json:"inspired_by_menu_item_name"`
	Status                 string     `json:"status"`
	GeneratedAt            time.Time  `json:"generated_at"`
	ReviewedAt             *time.Time `json:"reviewed_at"`
	// CreatedMenuItemID is set once this idea has been promoted to a real
	// menu_items row (see store.MenuIdeaStore.PromoteToMenuItem). Always
	// nil for "promotion"-kind ideas, which reference an existing item
	// rather than proposing a new one.
	CreatedMenuItemID *uuid.UUID `json:"created_menu_item_id"`

	IdeaCandidate
}

// PromptSignalRef identifies one adjacent signal that was actually shown to
// the LLM in a GenerateIdeas call — enough to trace a generated idea (or
// the lack of one) back to the specific TikTok video that informed it,
// without re-embedding the whole trend.Signal.
type PromptSignalRef struct {
	ID         uuid.UUID `json:"id"`
	ExternalID string    `json:"external_id"`
	Caption    string    `json:"caption"`
	ViewCount  int64     `json:"view_count"`
	URL        string    `json:"url"`
}

// CallLog records what one GenerateIdeas call actually did, for debugging
// and judging output quality after the fact — how many adjacent signals
// existed at all, which subset was actually sent to the LLM, the exact
// prompt, and the raw response. Built and returned on every code path
// (including graceful-degradation and hard-failure ones) so a menu item
// that produced zero ideas is just as inspectable as one that produced
// several.
type CallLog struct {
	MenuItemID   uuid.UUID `json:"menu_item_id"`
	MenuItemName string    `json:"menu_item_name"`
	CalledAt     time.Time `json:"called_at"`

	// AdjacentSignalCount is how many of SearchRelevant's results were
	// classified as tweak-candidates (caption doesn't name the item),
	// before capping to what's actually sent to the LLM. Zero means
	// either none were found, or embed/search failed before a count
	// could be known (see Error).
	AdjacentSignalCount int `json:"adjacent_signal_count"`
	// PromotionSignalCount is how many of SearchRelevant's results were
	// classified as promotion evidence (caption names the item) — the
	// evidence behind a "promotion" idea, if one was produced. Zero means
	// none found (or the search itself failed — see Error).
	PromotionSignalCount int `json:"promotion_signal_count"`
	// PromptSignals is the subset of tweak-candidate signals actually
	// rendered into the prompt (see maxPromptSignals) — the "which ones"
	// the LLM based its ideas on.
	PromptSignals []PromptSignalRef `json:"prompt_signals"`

	// PromptText and RawResponse are empty when the call never reached
	// the LLM (e.g. embed/search failure, zero adjacent signals).
	PromptText  string `json:"prompt_text"`
	RawResponse string `json:"raw_response"`

	IdeaCount int `json:"idea_count"`

	// Error is a human-readable note on what degraded or failed this
	// call (embed failure, search failure, LLM error, malformed JSON).
	// Empty on a fully successful call, including the "zero adjacent
	// signals found" case, which isn't itself an error.
	Error string `json:"error"`
}
