-- video_url lets the Menu Idea Agent (and any future consumer of the
-- trend_signals corpus) attach a link back to the actual TikTok video that
-- fed an idea, instead of just captions/hashtags. Nullable: rows ingested
-- before this migration have no URL on file, and Upsert leaves it
-- first-seen (like caption/hashtags/embedding) rather than refreshing it.
ALTER TABLE trend_signals ADD COLUMN video_url TEXT;

-- top_comments captures a handful of comment texts + like counts at
-- ingestion time, per video -- richer evidence than caption/hashtags alone.
-- Nothing consumes this yet (same "capture now, use later" pattern as
-- video_url); it's ingestion-layer enrichment for future sentiment/entity
-- analysis. First-seen only, like caption/hashtags/embedding/video_url --
-- Upsert never refreshes it on conflict.
ALTER TABLE trend_signals ADD COLUMN top_comments JSONB NOT NULL DEFAULT '[]';
