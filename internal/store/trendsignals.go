package store

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/pgvector/pgvector-go"

	"fbperformance/internal/agents/menuidea"
	"fbperformance/internal/agents/trend"
)

// similarityCutoff bounds SearchSimilar to genuinely relevant matches — a
// conservative starting value (cosine distance; lower is more similar),
// tunable once there's a real corpus to calibrate against. Without a
// cutoff, VideoCount would read ~limit for nearly every query regardless of
// whether the item has any real matching signal.
const similarityCutoff = 0.35

// relevanceCutoff is the outer bound of "meaningfully related to this menu
// item" for SearchRelevant. Deliberately its own literal (0.35), NOT an
// alias of similarityCutoff: they happen to share a value today, but Trend
// Agent's cutoff and this agent's relevance cutoff are independent tuning
// knobs. Aliasing them would mean retuning Trend's similarityCutoff
// silently shifts the Menu Idea Agent's search too — keep them separate
// even though they start equal.
//
// There is deliberately no second, tighter "near-duplicate" distance band
// below this one. An earlier version tried that (excluding distance <
// 0.15 as "essentially the same item"), but real embeddings don't support
// the distinction: a real TikTok video literally captioned "Birria
// Tacos!!" landed at distance 0.32, *between* two videos about genuinely
// different birria fusion dishes (0.316 and 0.33). Cosine distance on a
// short "Name (Category)" query measures topical closeness, not "is this
// literally the item" — that only shows up in the caption text. See
// menuidea.Agent.GenerateIdeas, which classifies promotion-vs-tweak by
// checking whether the item's name appears in the caption instead.
const relevanceCutoff = 0.35

// TrendSignalStore persists and semantically searches the TikTok
// trend-signal corpus. It implements trend.SignalSearcher.
type TrendSignalStore struct {
	pool *pgxpool.Pool
}

func NewTrendSignalStore(pool *pgxpool.Pool) *TrendSignalStore {
	return &TrendSignalStore{pool: pool}
}

// Upsert inserts a signal, or refreshes its mutable engagement counters if
// one with the same (source, external_id) already exists. Caption,
// hashtags, embedding, video_url, top_comments, and posted_at are left as
// first-seen — only view/like/comment/share counts and ingested_at are
// refreshed, since those genuinely grow over a video's lifetime.
func (s *TrendSignalStore) Upsert(ctx context.Context, signal trend.Signal, embedding []float32) error {
	hashtags, err := json.Marshal(signal.Hashtags)
	if err != nil {
		return fmt.Errorf("store: marshal hashtags: %w", err)
	}
	topComments, err := json.Marshal(signal.TopComments)
	if err != nil {
		return fmt.Errorf("store: marshal top comments: %w", err)
	}

	_, err = s.pool.Exec(ctx, `
		INSERT INTO trend_signals (
			source, external_id, caption, hashtags, embedding, video_url, top_comments,
			view_count, like_count, comment_count, share_count, posted_at
		) VALUES (
			$1, $2, $3, $4::jsonb, $5, $6, $7::jsonb,
			$8, $9, $10, $11, $12
		)
		ON CONFLICT (source, external_id) DO UPDATE SET
			view_count = EXCLUDED.view_count,
			like_count = EXCLUDED.like_count,
			comment_count = EXCLUDED.comment_count,
			share_count = EXCLUDED.share_count,
			ingested_at = CURRENT_TIMESTAMP`,
		signal.Source, signal.ExternalID, signal.Caption, string(hashtags), pgvector.NewVector(embedding), nullIfEmpty(signal.URL), string(topComments),
		signal.ViewCount, signal.LikeCount, signal.CommentCount, signal.ShareCount, signal.PostedAt,
	)
	if err != nil {
		return fmt.Errorf("store: upsert trend signal: %w", err)
	}
	return nil
}

// SearchSimilar returns signals whose embedding is closest to embedding
// (cosine distance), most-similar first, filtered to those posted at or
// after since and within similarityCutoff.
func (s *TrendSignalStore) SearchSimilar(ctx context.Context, embedding []float32, limit int, since time.Time) ([]trend.Signal, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT id, source, external_id, caption, hashtags, video_url, top_comments,
			view_count, like_count, comment_count, share_count, posted_at
		FROM trend_signals
		WHERE posted_at >= $1 AND embedding <=> $2 < $3
		ORDER BY embedding <=> $2
		LIMIT $4`,
		since, pgvector.NewVector(embedding), similarityCutoff, limit,
	)
	if err != nil {
		return nil, fmt.Errorf("store: search similar trend signals: %w", err)
	}
	defer rows.Close()

	var signals []trend.Signal
	for rows.Next() {
		var sig trend.Signal
		var hashtags []byte
		var videoURL *string
		var topComments []byte
		if err := rows.Scan(
			&sig.ID, &sig.Source, &sig.ExternalID, &sig.Caption, &hashtags, &videoURL, &topComments,
			&sig.ViewCount, &sig.LikeCount, &sig.CommentCount, &sig.ShareCount, &sig.PostedAt,
		); err != nil {
			return nil, fmt.Errorf("store: scan trend signal: %w", err)
		}
		if err := json.Unmarshal(hashtags, &sig.Hashtags); err != nil {
			return nil, fmt.Errorf("store: unmarshal hashtags: %w", err)
		}
		if err := json.Unmarshal(topComments, &sig.TopComments); err != nil {
			return nil, fmt.Errorf("store: unmarshal top comments: %w", err)
		}
		if videoURL != nil {
			sig.URL = *videoURL
		}
		signals = append(signals, sig)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("store: search similar trend signals: %w", err)
	}
	return signals, nil
}

// SearchRelevant returns signals whose embedding is meaningfully related to
// embedding (distance < relevanceCutoff), most-similar-first, filtered to
// those posted at or after since. Used by the Menu Idea Agent as the single
// evidence pool for one menu item; the agent itself classifies each result
// as "promotion" (caption names the item) or "tweak-candidate" (it
// doesn't) by text, not by a second distance band — see relevanceCutoff's
// doc comment for why. Returns menuidea.RelevantSignal directly (not a
// store-local type) so *TrendSignalStore satisfies menuidea.SignalSearcher
// with no adapter needed, matching how SearchSimilar already returns
// trend.Signal directly.
func (s *TrendSignalStore) SearchRelevant(ctx context.Context, embedding []float32, limit int, since time.Time) ([]menuidea.RelevantSignal, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT id, source, external_id, caption, hashtags, video_url, top_comments,
			view_count, like_count, comment_count, share_count, posted_at,
			embedding <=> $2 AS distance
		FROM trend_signals
		WHERE posted_at >= $1 AND embedding <=> $2 < $3
		ORDER BY embedding <=> $2
		LIMIT $4`,
		since, pgvector.NewVector(embedding), relevanceCutoff, limit,
	)
	if err != nil {
		return nil, fmt.Errorf("store: search relevant trend signals: %w", err)
	}
	defer rows.Close()

	var results []menuidea.RelevantSignal
	for rows.Next() {
		var sig trend.Signal
		var hashtags []byte
		var videoURL *string
		var topComments []byte
		var distance float64
		if err := rows.Scan(
			&sig.ID, &sig.Source, &sig.ExternalID, &sig.Caption, &hashtags, &videoURL, &topComments,
			&sig.ViewCount, &sig.LikeCount, &sig.CommentCount, &sig.ShareCount, &sig.PostedAt,
			&distance,
		); err != nil {
			return nil, fmt.Errorf("store: scan trend signal: %w", err)
		}
		if err := json.Unmarshal(hashtags, &sig.Hashtags); err != nil {
			return nil, fmt.Errorf("store: unmarshal hashtags: %w", err)
		}
		if err := json.Unmarshal(topComments, &sig.TopComments); err != nil {
			return nil, fmt.Errorf("store: unmarshal top comments: %w", err)
		}
		if videoURL != nil {
			sig.URL = *videoURL
		}
		results = append(results, menuidea.RelevantSignal{Signal: sig, Distance: distance})
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("store: search relevant trend signals: %w", err)
	}
	return results, nil
}
