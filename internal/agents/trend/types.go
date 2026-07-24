package trend

import (
	"context"
	"time"

	"github.com/google/uuid"

	"fbperformance/internal/services/llm"
)

// EmbeddingDimensions is the single source of truth for the embedding
// vector width used both when ingesting trend signals and when embedding a
// query at Analyze time. It must match the `vector(...)` column width in
// the trend_signals migration.
const EmbeddingDimensions = 768

// Generator is the subset of the LLM client the Trend Agent depends on for
// text generation (narrative synthesis, sentiment/related-trend classification).
type Generator interface {
	Generate(ctx context.Context, model, prompt string, opts llm.GenerateOptions) (string, error)
}

// Embedder is the subset of the LLM client the Trend Agent depends on to
// embed a query concept for semantic retrieval.
type Embedder interface {
	Embed(ctx context.Context, model, text string, opts llm.EmbedOptions) ([]float32, error)
}

// Signal is a single ingested TikTok video from the trend-signal corpus,
// the unit semantic search operates over.
type Signal struct {
	ID           uuid.UUID
	Source       string
	ExternalID   string
	Caption      string
	Hashtags     []string
	ViewCount    int64
	LikeCount    int64
	CommentCount int64
	ShareCount   int64
	PostedAt     time.Time
}

// SignalSearcher is the subset of trend-signal storage the Trend Agent
// depends on for semantic retrieval.
type SignalSearcher interface {
	// SearchSimilar returns signals whose embedding is closest to embedding,
	// most-similar first, filtered to those posted at or after since.
	SearchSimilar(ctx context.Context, embedding []float32, limit int, since time.Time) ([]Signal, error)
}

// SnapshotReader is the subset of trend-snapshot storage the Trend Agent
// depends on to compute growth and ground its analysis in real data.
type SnapshotReader interface {
	// Recent returns up to limit snapshots for the given menu item, most
	// recent first.
	Recent(ctx context.Context, menuItemID uuid.UUID, limit int) ([]Snapshot, error)
}

// SnapshotWriter is the subset of trend-snapshot storage the Trend Agent
// depends on to persist the snapshot it just computed.
type SnapshotWriter interface {
	Save(ctx context.Context, snap Snapshot) error
}

// SnapshotStore is the full trend-snapshot storage dependency Analyze
// needs: it reads the prior snapshot (for growth) and writes the new one.
type SnapshotStore interface {
	SnapshotReader
	SnapshotWriter
}

// Input is the menu item the Trend Agent analyzes.
type Input struct {
	MenuItemID uuid.UUID
	ItemName   string
}

// Snapshot is a point-in-time capture of a menu item's TikTok signal,
// aggregated from a semantic search over the trend-signal corpus. Written
// and read by Analyze on every call, to compute growth over time.
type Snapshot struct {
	ID             uuid.UUID
	MenuItemID     uuid.UUID
	CapturedAt     time.Time
	VideoCount     int
	TotalViews     int64
	TotalLikes     int64
	TotalComments  int64
	TotalShares    int64
	EngagementRate float64
	SentimentLabel string
	SentimentScore float64
	TopHashtags    []string
	RelatedTrends  []string
	Summary        string
}

// Result is the Trend Agent's output for a single Analyze call: the
// qualitative narrative plus the structured metrics it was grounded in, so
// downstream consumers (the Manager Agent, API clients) can reason over the
// numbers directly rather than parsing prose.
type Result struct {
	Summary string

	VideoCount     int
	TotalViews     int64
	TotalLikes     int64
	TotalComments  int64
	TotalShares    int64
	EngagementRate float64

	SentimentLabel string
	SentimentScore float64

	// GrowthRatePct is nil when fewer than 2 snapshots exist, or when the
	// prior snapshot's TotalViews is 0 (no baseline to compare against).
	GrowthRatePct *float64
	// GrowthPeriodHours is the elapsed time between the two most recent
	// snapshots used to compute GrowthRatePct. Populated whenever 2
	// snapshots exist, even if GrowthRatePct itself is nil.
	GrowthPeriodHours float64

	TopHashtags   []string
	RelatedTrends []string
}
