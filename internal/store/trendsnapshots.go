package store

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"fbperformance/internal/agents/trend"
)

// TrendSnapshotStore persists and reads back trend.Snapshot rows. It
// implements trend.SnapshotReader.
type TrendSnapshotStore struct {
	pool *pgxpool.Pool
}

func NewTrendSnapshotStore(pool *pgxpool.Pool) *TrendSnapshotStore {
	return &TrendSnapshotStore{pool: pool}
}

// Save inserts a new trend snapshot row. Duplicate snapshots for the same
// menu item (e.g. from an overlapping sync run) are accepted, not rejected
// — growth-rate calc reads the two most recent rows by captured_at, so
// duplicates don't break correctness, they just waste storage.
func (s *TrendSnapshotStore) Save(ctx context.Context, snap trend.Snapshot) error {
	topHashtags, err := json.Marshal(snap.TopHashtags)
	if err != nil {
		return fmt.Errorf("store: marshal top_hashtags: %w", err)
	}
	relatedTrends, err := json.Marshal(snap.RelatedTrends)
	if err != nil {
		return fmt.Errorf("store: marshal related_trends: %w", err)
	}

	_, err = s.pool.Exec(ctx, `
		INSERT INTO trend_snapshots (
			menu_item_id, captured_at, video_count,
			total_views, total_likes, total_comments, total_shares,
			engagement_rate, sentiment_label, sentiment_score,
			top_hashtags, related_trends, summary
		) VALUES (
			$1, $2, $3,
			$4, $5, $6, $7,
			$8, $9, $10,
			$11::jsonb, $12::jsonb, $13
		)`,
		snap.MenuItemID, snap.CapturedAt, snap.VideoCount,
		snap.TotalViews, snap.TotalLikes, snap.TotalComments, snap.TotalShares,
		snap.EngagementRate, snap.SentimentLabel, snap.SentimentScore,
		string(topHashtags), string(relatedTrends), snap.Summary,
	)
	if err != nil {
		return fmt.Errorf("store: save trend snapshot: %w", err)
	}
	return nil
}

// Recent returns up to limit trend snapshots for the given menu item, most
// recent first.
func (s *TrendSnapshotStore) Recent(ctx context.Context, menuItemID uuid.UUID, limit int) ([]trend.Snapshot, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT id, menu_item_id, captured_at, video_count,
			total_views, total_likes, total_comments, total_shares,
			engagement_rate, sentiment_label, sentiment_score,
			top_hashtags, related_trends, summary
		FROM trend_snapshots
		WHERE menu_item_id = $1
		ORDER BY captured_at DESC
		LIMIT $2`,
		menuItemID, limit,
	)
	if err != nil {
		return nil, fmt.Errorf("store: read recent trend snapshots: %w", err)
	}
	defer rows.Close()

	var snapshots []trend.Snapshot
	for rows.Next() {
		var snap trend.Snapshot
		var topHashtags, relatedTrends []byte
		var summary *string
		if err := rows.Scan(
			&snap.ID, &snap.MenuItemID, &snap.CapturedAt, &snap.VideoCount,
			&snap.TotalViews, &snap.TotalLikes, &snap.TotalComments, &snap.TotalShares,
			&snap.EngagementRate, &snap.SentimentLabel, &snap.SentimentScore,
			&topHashtags, &relatedTrends, &summary,
		); err != nil {
			return nil, fmt.Errorf("store: scan trend snapshot: %w", err)
		}
		if err := json.Unmarshal(topHashtags, &snap.TopHashtags); err != nil {
			return nil, fmt.Errorf("store: unmarshal top_hashtags: %w", err)
		}
		if err := json.Unmarshal(relatedTrends, &snap.RelatedTrends); err != nil {
			return nil, fmt.Errorf("store: unmarshal related_trends: %w", err)
		}
		if summary != nil {
			snap.Summary = *summary
		}
		snapshots = append(snapshots, snap)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("store: read recent trend snapshots: %w", err)
	}
	return snapshots, nil
}
