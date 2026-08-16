CREATE TABLE menu_idea_agent_calls (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    batch_run_id UUID NOT NULL,
    menu_item_id UUID REFERENCES menu_items(id),
    menu_item_name VARCHAR(255) NOT NULL,
    called_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    adjacent_signal_count INT NOT NULL DEFAULT 0,
    promotion_signal_count INT NOT NULL DEFAULT 0,
    prompt_signals JSONB NOT NULL DEFAULT '[]',
    prompt_text TEXT,
    raw_response TEXT,
    idea_count INT NOT NULL DEFAULT 0,
    error TEXT
);

-- One row per menu item per batch run of cmd/menu-idea-gen, regardless of
-- outcome: how many adjacent TikTok signals existed (adjacent_signal_count),
-- which ones were actually shown to the LLM (prompt_signals -- id,
-- external_id, caption, view_count), the exact prompt sent (prompt_text),
-- and the raw LLM response (raw_response). Lets a human judge whether the
-- agent is looking at the right evidence and using it well, not just see
-- the final idea. error is set when embed/search/generate/parse degraded
-- or failed for that item; NULL on a fully successful call (including the
-- "zero adjacent signals found" case, which isn't itself an error).

-- promotion_signal_count records how many near-duplicate signals
-- SearchNearDuplicate found for this call (see menuidea.CallLog), alongside
-- adjacent_signal_count for the tweak-band search.

CREATE INDEX idx_menu_idea_agent_calls_batch_run_id ON menu_idea_agent_calls (batch_run_id);
CREATE INDEX idx_menu_idea_agent_calls_menu_item_id ON menu_idea_agent_calls (menu_item_id);
CREATE INDEX idx_menu_idea_agent_calls_called_at ON menu_idea_agent_calls (called_at DESC);
