// Package trendingest holds the TikTok hashtag-sweep logic shared by
// cmd/trend-ingest (the manually/externally-scheduled CLI) and the API
// server's approve-triggered background run (see Service.TriggerAsync) —
// one implementation of "sweep these hashtags into trend_signals", two
// callers.
package trendingest

import (
	"context"
	"log"
	"sync"
	"time"

	"fbperformance/internal/agents/trend"
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

// Ingester sweeps a list of hashtags via Apify/TikTok, embeds each video's
// caption, and upserts them into the trend_signals corpus.
type Ingester struct {
	tiktok     *tiktok.Client
	llm        *llm.Client
	signals    *store.TrendSignalStore
	embedModel string
}

func New(tiktokClient *tiktok.Client, llmClient *llm.Client, signals *store.TrendSignalStore, embedModel string) *Ingester {
	return &Ingester{tiktok: tiktokClient, llm: llmClient, signals: signals, embedModel: embedModel}
}

// Summary is the aggregate result of one Run call across all hashtags.
type Summary struct {
	HashtagsSwept  int
	VideosUpserted int
	// HashtagsFailed counts hashtags whose FetchByHashtags call itself
	// errored (auth, network, rate limit) — a whole-hashtag failure,
	// distinct from a single bad video's embed/upsert error (which is
	// still best-effort-skipped and doesn't count here). Callers use this
	// to tell "ran but found nothing" apart from "every sweep errored out"
	// — see Service.TriggerAsync, which reports the latter as "failed"
	// instead of a silently-successful zero-result "completed".
	HashtagsFailed int
	// LastFetchError is one representative fetch error (whichever hashtag
	// happened to fail last), surfaced to the frontend when every hashtag
	// failed — not meant to enumerate every failure, just to give a human
	// enough to act on (e.g. "401: token not provided" points straight at
	// a bad APIFY_API_TOKEN).
	LastFetchError error
}

// Run sweeps every hashtag concurrently (bounded by ingestConcurrency) and
// blocks until all sweeps finish. Per-video failures (embed/upsert) are
// logged and skipped rather than failing the whole run — one bad video
// shouldn't abort an otherwise-healthy sweep. Per-hashtag fetch failures
// are also logged and skipped the same way, but are additionally counted
// in Summary.HashtagsFailed so a total failure (e.g. a bad Apify token) is
// distinguishable from a real zero-result run.
func (in *Ingester) Run(ctx context.Context, hashtags []string) Summary {
	var upserted, failed int64
	var mu sync.Mutex
	var lastFetchErr error

	sem := make(chan struct{}, ingestConcurrency)
	var wg sync.WaitGroup
	for _, hashtag := range hashtags {
		wg.Add(1)
		go func(hashtag string) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()
			n, fetchErr := in.sweepHashtag(ctx, hashtag)
			mu.Lock()
			upserted += int64(n)
			if fetchErr != nil {
				failed++
				lastFetchErr = fetchErr
			}
			mu.Unlock()
		}(hashtag)
	}
	wg.Wait()

	return Summary{
		HashtagsSwept:  len(hashtags),
		VideosUpserted: int(upserted),
		HashtagsFailed: int(failed),
		LastFetchError: lastFetchErr,
	}
}

// sweepHashtag returns the number of videos upserted, and a non-nil error
// if the initial FetchByHashtags call itself failed (the whole-hashtag
// failure mode) — per-video embed/upsert errors are still logged and
// skipped without being returned here, matching Run's doc comment.
func (in *Ingester) sweepHashtag(ctx context.Context, hashtag string) (int, error) {
	start := time.Now()

	videos, err := in.tiktok.FetchByHashtags(ctx, []string{hashtag}, resultsPerPage)
	if err != nil {
		log.Printf("trendingest: hashtag #%s: fetch videos: %v", hashtag, err)
		return 0, err
	}

	var upserted int
	for _, video := range videos {
		embedding, err := in.llm.Embed(ctx, in.embedModel, video.Caption, llm.EmbedOptions{
			TaskType:             "RETRIEVAL_DOCUMENT",
			OutputDimensionality: trend.EmbeddingDimensions,
		})
		if err != nil {
			log.Printf("trendingest: hashtag #%s: embed video %s: %v", hashtag, video.ID, err)
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
		if err := in.signals.Upsert(ctx, signal, embedding); err != nil {
			log.Printf("trendingest: hashtag #%s: upsert video %s: %v", hashtag, video.ID, err)
			continue
		}
		upserted++
	}

	log.Printf("trendingest: hashtag #%s: %d/%d videos upserted (%s)", hashtag, upserted, len(videos), time.Since(start).Round(time.Second))
	return upserted, nil
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
