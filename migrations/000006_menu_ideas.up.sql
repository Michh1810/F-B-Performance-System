CREATE TABLE menu_ideas (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    batch_run_id UUID NOT NULL,
    idea_name VARCHAR(255) NOT NULL,
    idea_description TEXT NOT NULL,
    suggested_category VARCHAR(255),
    rationale TEXT NOT NULL,
    inspired_by_menu_item_id UUID REFERENCES menu_items(id),
    inspired_by_menu_item_name VARCHAR(255) NOT NULL,
    source_hashtags JSONB NOT NULL DEFAULT '[]',
    source_signal_count INT NOT NULL DEFAULT 0,
    source_total_views BIGINT NOT NULL DEFAULT 0,
    source_video_urls JSONB NOT NULL DEFAULT '[]',
    status VARCHAR(32) NOT NULL DEFAULT 'new',
    kind VARCHAR(32) NOT NULL DEFAULT 'tweak',
    created_menu_item_id UUID REFERENCES menu_items(id),
    generated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    reviewed_at TIMESTAMPTZ
);

-- status is one of: new (default, unreviewed) | reviewed (a human looked at
-- it, no verdict implied) | dismissed (rejected) | promoted (a staff
-- bookmark/shortlist marker -- no side effects, does not create a menu_items
-- row or trigger anything else; purely a label a human can filter on).

-- source_video_urls carries the evidence-clip links for a generated idea
-- through to the review API, deduped in the same first-seen order as
-- source_hashtags.

-- kind distinguishes an LLM-proposed new item ('tweak', from adjacent-band
-- trend evidence) from a deterministically-synthesized call to feature an
-- existing item that's trending as-is ('promotion', from near-duplicate-band
-- evidence).

-- created_menu_item_id links a promoted "tweak" idea to the real menu_items
-- row it produced, so PromoteToMenuItem is idempotent (re-promoting an
-- already-promoted idea returns the existing link instead of creating a
-- second row). NULL for ideas never promoted, and for "promotion"-kind
-- ideas (which reference an existing item, so there's nothing new to
-- create).

CREATE INDEX idx_menu_ideas_status ON menu_ideas (status);
CREATE INDEX idx_menu_ideas_generated_at ON menu_ideas (generated_at DESC);
CREATE INDEX idx_menu_ideas_kind ON menu_ideas (kind);
