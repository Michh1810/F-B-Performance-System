-- restaurant_profile is a singleton table: always exactly one row, keyed on
-- a fixed id (see store.restaurantProfileSingletonID), rather than a
-- separate "current row" pointer. Holds a free-text description of this
-- restaurant's cuisine/style, entered via the frontend, used to ground
-- trend_hashtag_suggestions generation in what this specific restaurant
-- actually sells.
CREATE TABLE restaurant_profile (
    id UUID PRIMARY KEY,
    description TEXT NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- One row per hashtag-suggestion generation call: source_description is a
-- denormalized copy of the profile description at generation time (so a
-- later profile edit doesn't retroactively change what an old suggestion
-- was based on). status starts "pending" and is set to "approved" or
-- "rejected" by a human reviewing it; cmd/trend-ingest sweeps whichever
-- approved suggestion's hashtags were reviewed most recently.
CREATE TABLE trend_hashtag_suggestions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    source_description TEXT NOT NULL,
    hashtags JSONB NOT NULL,
    status VARCHAR(32) NOT NULL DEFAULT 'pending',
    generated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    reviewed_at TIMESTAMPTZ
);

CREATE INDEX idx_trend_hashtag_suggestions_status ON trend_hashtag_suggestions (status);
CREATE INDEX idx_trend_hashtag_suggestions_generated_at ON trend_hashtag_suggestions (generated_at DESC);
