// Command trend-ingest is a one-shot batch job: for each hashtag in a
// curated, menu-item-agnostic list, it fetches matching TikTok videos via
// Apify, embeds each caption, and upserts them into the trend_signals
// corpus. It is meant to be invoked by an external scheduler (cron, a
// Kubernetes CronJob, etc.) — this binary does not self-schedule, and it
// does not guard against overlapping invocations; that is the scheduler's
// responsibility (see README).
//
// The sweep logic itself lives in internal/services/trendingest, shared
// with the API server's approve-triggered background run (see
// TrendHashtagsHandler.UpdateStatus) — this binary is one of two callers.
//
// Trend coverage is bounded by this hashtag list: a real trend that's never
// swept under one of these tags won't be in the corpus at all, no matter
// how well it would semantically match a query. This is semantic matching
// over a curated corpus, not open-ended discovery across all of TikTok.
package main

import (
	"context"
	"log"

	"github.com/joho/godotenv"

	"fbperformance/internal/config"
	"fbperformance/internal/services/llm"
	"fbperformance/internal/services/tiktok"
	"fbperformance/internal/services/trendingest"
	"fbperformance/internal/store"
)

func main() {
	_ = godotenv.Load()

	cfg := config.Load()
	ctx := context.Background()

	pool, err := store.Connect(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("connect database: %v", err)
	}
	defer pool.Close()

	signalStore := store.NewTrendSignalStore(pool)
	hashtagSuggestionStore := store.NewTrendHashtagSuggestionStore(pool)
	tiktokClient := tiktok.NewClient(cfg.ApifyAPIToken, cfg.ApifyActorID)
	llmClient := llm.NewClient(cfg.GeminiAPIKey)
	ingester := trendingest.New(tiktokClient, llmClient, signalStore, cfg.GeminiEmbedModel)

	hashtags := cfg.TrendIngestHashtags
	if approved, err := hashtagSuggestionStore.LatestApproved(ctx); err != nil {
		log.Printf("trend-ingest: load approved hashtags: %v (falling back to configured default)", err)
	} else if len(approved) > 0 {
		hashtags = approved
		log.Printf("trend-ingest: using %d approved hashtags from the restaurant-profile workflow", len(hashtags))
	}

	log.Printf("trend-ingest: sweeping %d hashtags", len(hashtags))
	summary := ingester.Run(ctx, hashtags)
	log.Printf("trend-ingest: done, %d videos upserted across %d hashtags", summary.VideosUpserted, summary.HashtagsSwept)
}
