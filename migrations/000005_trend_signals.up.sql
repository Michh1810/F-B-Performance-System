CREATE EXTENSION IF NOT EXISTS vector;

-- embedding dimensionality (768) must match trend.EmbeddingDimensions in Go.
CREATE TABLE trend_signals (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    source VARCHAR(32) NOT NULL,
    external_id VARCHAR(128) NOT NULL,
    caption TEXT NOT NULL,
    hashtags JSONB NOT NULL DEFAULT '[]',
    embedding vector(768) NOT NULL,
    view_count BIGINT NOT NULL DEFAULT 0,
    like_count BIGINT NOT NULL DEFAULT 0,
    comment_count BIGINT NOT NULL DEFAULT 0,
    share_count BIGINT NOT NULL DEFAULT 0,
    posted_at TIMESTAMPTZ,
    ingested_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE (source, external_id)
);

CREATE INDEX idx_trend_signals_embedding ON trend_signals USING hnsw (embedding vector_cosine_ops);
CREATE INDEX idx_trend_signals_posted_at ON trend_signals (posted_at DESC);
