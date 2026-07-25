CREATE TABLE trend_snapshots (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    menu_item_id UUID NOT NULL REFERENCES menu_items(id),
    captured_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    video_count INTEGER NOT NULL,
    total_views BIGINT NOT NULL,
    total_likes BIGINT NOT NULL,
    total_comments BIGINT NOT NULL,
    total_shares BIGINT NOT NULL,
    engagement_rate DECIMAL(6, 4) NOT NULL,
    sentiment_label VARCHAR(16) NOT NULL,
    sentiment_score DECIMAL(4, 3) NOT NULL,
    top_hashtags JSONB NOT NULL DEFAULT '[]',
    related_trends JSONB NOT NULL DEFAULT '[]',
    summary TEXT
);

CREATE INDEX idx_trend_snapshots_menu_item_captured ON trend_snapshots (menu_item_id, captured_at DESC);
