// Package trend implements the Trend Agent: given a menu item (existing or
// hypothetical), it embeds the item as a semantic query, searches a
// TikTok trend-signal corpus for the closest matches, aggregates and
// classifies them, computes growth against the item's prior evaluation, and
// produces a market-viability narrative grounded in those real numbers.
package trend

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"time"

	"fbperformance/internal/services/llm"
)

const (
	// signalSearchLimit bounds how many semantically-matched signals a
	// single Analyze call considers.
	signalSearchLimit = 20
	// topSignalsForClassification bounds how many of those matched signals
	// are sent to Gemini for sentiment/related-trend classification,
	// keeping the prompt compact and cheap.
	topSignalsForClassification = 15

	embedTimeout     = 5 * time.Second
	classifyTimeout  = 10 * time.Second
	narrativeTimeout = 15 * time.Second
)

type Agent struct {
	client       Generator
	embedder     Embedder
	signals      SignalSearcher
	snapshots    SnapshotStore
	model        string
	embedModel   string
	lookbackDays int
}

func NewAgent(client Generator, embedder Embedder, signals SignalSearcher, snapshots SnapshotStore, model, embedModel string, lookbackDays int) *Agent {
	return &Agent{
		client:       client,
		embedder:     embedder,
		signals:      signals,
		snapshots:    snapshots,
		model:        model,
		embedModel:   embedModel,
		lookbackDays: lookbackDays,
	}
}

// Analyze embeds the item as a query, semantically searches the trend-signal
// corpus, aggregates and classifies the results, computes growth against the
// item's most recent prior evaluation, and returns a narrative grounded in
// those numbers. It both reads the prior snapshot and writes a new one on
// every call — there is no separate offline ingestion step for snapshots.
func (a *Agent) Analyze(ctx context.Context, in Input) (Result, error) {
	queryEmbedding, err := a.embedQuery(ctx, in.ItemName)
	if err != nil {
		return noSignalResult(), nil
	}

	since := time.Now().AddDate(0, 0, -a.lookbackDays)
	signals, err := a.signals.SearchSimilar(ctx, queryEmbedding, signalSearchLimit, since)
	if err != nil || len(signals) == 0 {
		return noSignalResult(), nil
	}

	snapshot := aggregateSignals(signals)
	snapshot.MenuItemID = in.MenuItemID
	snapshot.CapturedAt = time.Now().UTC()
	a.classify(ctx, signals, &snapshot)

	previous, err := a.snapshots.Recent(ctx, in.MenuItemID, 1)
	if err != nil {
		return Result{}, fmt.Errorf("trend agent: read previous snapshot: %w", err)
	}

	var growthPct *float64
	var growthPeriodHours float64
	if len(previous) >= 1 {
		growthPeriodHours = snapshot.CapturedAt.Sub(previous[0].CapturedAt).Hours()
		if previous[0].TotalViews > 0 {
			pct := float64(snapshot.TotalViews-previous[0].TotalViews) / float64(previous[0].TotalViews) * 100
			growthPct = &pct
		}
	}

	narrativeCtx, cancel := context.WithTimeout(ctx, narrativeTimeout)
	narrative, err := a.client.Generate(narrativeCtx, a.model, buildPrompt(in, snapshot, growthPct, growthPeriodHours), llm.GenerateOptions{
		SystemPrompt: systemPrompt,
	})
	cancel()
	if err != nil {
		return Result{}, fmt.Errorf("trend agent: %w", err)
	}
	snapshot.Summary = narrative

	if err := a.snapshots.Save(ctx, snapshot); err != nil {
		return Result{}, fmt.Errorf("trend agent: save snapshot: %w", err)
	}

	return Result{
		Summary:           narrative,
		VideoCount:        snapshot.VideoCount,
		TotalViews:        snapshot.TotalViews,
		TotalLikes:        snapshot.TotalLikes,
		TotalComments:     snapshot.TotalComments,
		TotalShares:       snapshot.TotalShares,
		EngagementRate:    snapshot.EngagementRate,
		SentimentLabel:    snapshot.SentimentLabel,
		SentimentScore:    snapshot.SentimentScore,
		GrowthRatePct:     growthPct,
		GrowthPeriodHours: growthPeriodHours,
		TopHashtags:       snapshot.TopHashtags,
		RelatedTrends:     snapshot.RelatedTrends,
	}, nil
}

// embedQuery embeds text as a search query (not a document — see
// llm.EmbedOptions.TaskType) for semantic retrieval against the corpus.
func (a *Agent) embedQuery(ctx context.Context, text string) ([]float32, error) {
	embedCtx, cancel := context.WithTimeout(ctx, embedTimeout)
	defer cancel()
	return a.embedder.Embed(embedCtx, a.embedModel, text, llm.EmbedOptions{
		TaskType:             "RETRIEVAL_QUERY",
		OutputDimensionality: EmbeddingDimensions,
	})
}

// classification is the JSON shape requested from the LLM in classify.
type classification struct {
	SentimentLabel string   `json:"sentiment_label"`
	SentimentScore float64  `json:"sentiment_score"`
	RelatedTrends  []string `json:"related_food_trends"`
}

// classify makes one Gemini call to label sentiment and infer related food
// trends from the top-matched signals, writing the result onto snapshot. On
// any failure (call error or malformed JSON) it degrades gracefully to
// "unknown" sentiment rather than failing the whole request — this runs
// inline in a live HTTP request now, not a skippable batch job.
func (a *Agent) classify(ctx context.Context, signals []Signal, snapshot *Snapshot) {
	classifyCtx, cancel := context.WithTimeout(ctx, classifyTimeout)
	defer cancel()

	text, err := a.client.Generate(classifyCtx, a.model, buildSnapshotPrompt(selectTopSignals(signals, topSignalsForClassification)), llm.GenerateOptions{
		SystemPrompt: snapshotSystemPrompt,
		JSONMode:     true,
	})
	if err == nil {
		var c classification
		if json.Unmarshal([]byte(text), &c) == nil {
			snapshot.SentimentLabel = c.SentimentLabel
			snapshot.SentimentScore = c.SentimentScore
			snapshot.RelatedTrends = c.RelatedTrends
		}
	}
	if snapshot.SentimentLabel == "" {
		snapshot.SentimentLabel = "unknown"
	}
	if snapshot.RelatedTrends == nil {
		snapshot.RelatedTrends = []string{}
	}
}

// noSignalResult is returned when the corpus has nothing relevant for this
// item — including when embedding or searching itself fails, which is
// treated identically to "found nothing" rather than as a request failure.
func noSignalResult() Result {
	return Result{
		Summary:        "No relevant TikTok signal found for this item.",
		SentimentLabel: "unknown",
	}
}

// aggregateSignals computes all quantitative metrics (counts, engagement
// rate, top hashtags) from a batch of matched signals in Go — no LLM
// involved, no token cost. MenuItemID, CapturedAt, and the
// classification/summary fields are populated by the caller.
func aggregateSignals(signals []Signal) Snapshot {
	snapshot := Snapshot{VideoCount: len(signals)}

	hashtagCounts := make(map[string]int)
	var hashtagOrder []string
	for _, s := range signals {
		snapshot.TotalViews += s.ViewCount
		snapshot.TotalLikes += s.LikeCount
		snapshot.TotalComments += s.CommentCount
		snapshot.TotalShares += s.ShareCount
		for _, h := range s.Hashtags {
			if h == "" {
				continue
			}
			if hashtagCounts[h] == 0 {
				hashtagOrder = append(hashtagOrder, h)
			}
			hashtagCounts[h]++
		}
	}
	if snapshot.TotalViews > 0 {
		snapshot.EngagementRate = float64(snapshot.TotalLikes+snapshot.TotalComments+snapshot.TotalShares) / float64(snapshot.TotalViews)
	}
	snapshot.TopHashtags = topHashtagsByFrequency(hashtagOrder, hashtagCounts, 10)

	return snapshot
}

// selectTopSignals returns up to n signals sorted by view count descending,
// without mutating the order of the caller's slice.
func selectTopSignals(signals []Signal, n int) []Signal {
	sorted := make([]Signal, len(signals))
	copy(sorted, signals)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i].ViewCount > sorted[j].ViewCount })
	if len(sorted) > n {
		sorted = sorted[:n]
	}
	return sorted
}

// topHashtagsByFrequency returns up to n hashtags ordered by descending
// frequency, breaking ties by first-seen order.
func topHashtagsByFrequency(order []string, counts map[string]int, n int) []string {
	sorted := make([]string, len(order))
	copy(sorted, order)
	sort.SliceStable(sorted, func(i, j int) bool { return counts[sorted[i]] > counts[sorted[j]] })
	if len(sorted) > n {
		sorted = sorted[:n]
	}
	return sorted
}
