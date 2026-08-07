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
	hashtagSuggestionStore := store.NewTrendHashtagSuggestionStore(pool)
	tiktokClient := tiktok.NewClient(cfg.ApifyAPIToken, cfg.ApifyActorID)
	llmClient := llm.NewClient(cfg.GeminiAPIKey)

	hashtags := cfg.TrendIngestHashtags
	if approved, err := hashtagSuggestionStore.LatestApproved(ctx); err != nil {
		log.Printf("trend-ingest: load approved hashtags: %v (falling back to configured default)", err)
	} else if len(approved) > 0 {
		hashtags = approved
		log.Printf("trend-ingest: using %d approved hashtags from the restaurant-profile workflow", len(hashtags))
	}

	log.Printf("trend-ingest: sweeping %d hashtags", len(hashtags))

	sem := make(chan struct{}, ingestConcurrency)
	var wg sync.WaitGroup
	for _, hashtag := range hashtags {
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
			URL:          video.URL,
			TopComments:  toTopComments(video.Comments),
		}
		if err := signalStore.Upsert(ctx, signal, embedding); err != nil {
			log.Printf("trend-ingest: hashtag #%s: upsert video %s: %v", hashtag, video.ID, err)
			continue
		}
		upserted++
	}

	log.Printf("trend-ingest: hashtag #%s: %d/%d videos upserted (%s)", hashtag, upserted, len(videos), time.Since(start).Round(time.Second))
}

// toTopComments converts tiktok.Comment (the ingestion client's DTO) into
// trend.Comment (the corpus's DTO) — kept as separate types so the trend
// package doesn't depend on the tiktok package, same reasoning as
// trend.Signal not reusing tiktok.Video.
func toTopComments(comments []tiktok.Comment) []trend.Comment {
	out := make([]trend.Comment, len(comments))
	for i, c := range comments {
		out[i] = trend.Comment{Text: c.Text, DiggCount: c.DiggCount}
	}
	return out
}
