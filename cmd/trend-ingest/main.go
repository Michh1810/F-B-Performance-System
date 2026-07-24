// Command trend-ingest is a one-shot batch job: for each hashtag in a
// curated, menu-item-agnostic list, it fetches matching TikTok videos via
// Apify, embeds each caption, and upserts them into the trend_signals
// corpus. It is meant to be invoked by an external scheduler (cron, a
// Kubernetes CronJob, etc.) — this binary does not self-schedule, and it
// does not guard against overlapping invocations; that is the scheduler's
// responsibility (see README).
//
// Trend coverage is bounded by this hashtag list: a real trend that's never
// swept under one of these tags won't be in the corpus at all, no matter
// how well it would semantically match a query. This is semantic matching
// over a curated corpus, not open-ended discovery across all of TikTok.
package main

import (
	"context"
	"log"
	"sync"
	"time"

	"github.com/joho/godotenv"

	"fbperformance/internal/agents/trend"
	"fbperformance/internal/config"
	"fbperformance/internal/services/llm"
	"fbperformance/internal/services/tiktok"
	"fbperformance/internal/store"
)

// resultsPerPage bounds how many videos Apify returns per hashtag search.
const resultsPerPage = 10

// ingestConcurrency bounds how many hashtags are swept at once. Apify actor
// runs take 30-120s each, so sweeping sequentially would make even a
// modest hashtag list take many minutes.
const ingestConcurrency = 4

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
	tiktokClient := tiktok.NewClient(cfg.ApifyAPIToken, cfg.ApifyActorID)
	llmClient := llm.NewClient(cfg.GeminiAPIKey)

	log.Printf("trend-ingest: sweeping %d hashtags", len(cfg.TrendIngestHashtags))

	sem := make(chan struct{}, ingestConcurrency)
	var wg sync.WaitGroup
	for _, hashtag := range cfg.TrendIngestHashtags {
		wg.Add(1)
		go func() {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()
			sweepHashtag(ctx, tiktokClient, llmClient, signalStore, cfg.GeminiEmbedModel, hashtag)
		}()
	}
	wg.Wait()

	log.Print("trend-ingest: done")
}

func sweepHashtag(ctx context.Context, tiktokClient *tiktok.Client, llmClient *llm.Client, signalStore *store.TrendSignalStore, embedModel, hashtag string) {
	start := time.Now()

	videos, err := tiktokClient.FetchByHashtags(ctx, []string{hashtag}, resultsPerPage)
	if err != nil {
		log.Printf("trend-ingest: hashtag #%s: fetch videos: %v", hashtag, err)
		return
	}

	var upserted int
	for _, video := range videos {
		embedding, err := llmClient.Embed(ctx, embedModel, video.Caption, llm.EmbedOptions{
			TaskType:             "RETRIEVAL_DOCUMENT",
			OutputDimensionality: trend.EmbeddingDimensions,
		})
		if err != nil {
			log.Printf("trend-ingest: hashtag #%s: embed video %s: %v", hashtag, video.ID, err)
			continue
		}

		signal := trend.Signal{
			Source:       "tiktok",
			ExternalID:   video.ID,
			Caption:      video.Caption,
			Hashtags:     video.Hashtags,
			ViewCount:    video.PlayCount,
			LikeCount:    video.DiggCount,
			CommentCount: video.CommentCount,
			ShareCount:   video.ShareCount,
			PostedAt:     video.PostedAt,
		}
		if err := signalStore.Upsert(ctx, signal, embedding); err != nil {
			log.Printf("trend-ingest: hashtag #%s: upsert video %s: %v", hashtag, video.ID, err)
			continue
		}
		upserted++
	}

	log.Printf("trend-ingest: hashtag #%s: %d/%d videos upserted (%s)", hashtag, upserted, len(videos), time.Since(start).Round(time.Second))
}
