package store

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/pgvector/pgvector-go"

	"fbperformance/internal/agents/trend"
)

// similarityCutoff bounds SearchSimilar to genuinely relevant matches — a
// conservative starting value (cosine distance; lower is more similar),
// tunable once there's a real corpus to calibrate against. Without a
// cutoff, VideoCount would read ~limit for nearly every query regardless of
// whether the item has any real matching signal.
const similarityCutoff = 0.35

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
// hashtags, embedding, and posted_at are left as first-seen — only view/
// like/comment/share counts and ingested_at are refreshed, since those
// genuinely grow over a video's lifetime.
func (s *TrendSignalStore) Upsert(ctx context.Context, signal trend.Signal, embedding []float32) error {
	hashtags, err := json.Marshal(signal.Hashtags)
	if err != nil {
		return fmt.Errorf("store: marshal hashtags: %w", err)
	}

	_, err = s.pool.Exec(ctx, `
		INSERT INTO trend_signals (
			source, external_id, caption, hashtags, embedding,
			view_count, like_count, comment_count, share_count, posted_at
		) VALUES (
			$1, $2, $3, $4::jsonb, $5,
			$6, $7, $8, $9, $10
		)
		ON CONFLICT (source, external_id) DO UPDATE SET
			view_count = EXCLUDED.view_count,
			like_count = EXCLUDED.like_count,
			comment_count = EXCLUDED.comment_count,
			share_count = EXCLUDED.share_count,
			ingested_at = CURRENT_TIMESTAMP`,
		signal.Source, signal.ExternalID, signal.Caption, string(hashtags), pgvector.NewVector(embedding),
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
		SELECT id, source, external_id, caption, hashtags,
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
		if err := rows.Scan(
			&sig.ID, &sig.Source, &sig.ExternalID, &sig.Caption, &hashtags,
			&sig.ViewCount, &sig.LikeCount, &sig.CommentCount, &sig.ShareCount, &sig.PostedAt,
		); err != nil {
			return nil, fmt.Errorf("store: scan trend signal: %w", err)
		}
		if err := json.Unmarshal(hashtags, &sig.Hashtags); err != nil {
			return nil, fmt.Errorf("store: unmarshal hashtags: %w", err)
		}
		signals = append(signals, sig)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("store: search similar trend signals: %w", err)
	}
	return signals, nil
}
