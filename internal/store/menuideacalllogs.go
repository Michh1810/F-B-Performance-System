package store

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"fbperformance/internal/agents/menuidea"
)

// MenuIdeaCallLogStore persists one debug record per menu item per batch
// run of cmd/menu-idea-gen — see menuidea.CallLog for what's captured and
// why.
type MenuIdeaCallLogStore struct {
	pool *pgxpool.Pool
}

func NewMenuIdeaCallLogStore(pool *pgxpool.Pool) *MenuIdeaCallLogStore {
	return &MenuIdeaCallLogStore{pool: pool}
}

// Save persists one CallLog. It is deliberately tolerant of a zero-value
// MenuItemID never happening in practice (the batch job always calls this
// with a real menu item), so no special-casing is done here — a caller
// passing an invalid ID will simply get the FK violation back as an error.
func (s *MenuIdeaCallLogStore) Save(ctx context.Context, batchRunID uuid.UUID, log menuidea.CallLog) error {
	promptSignals, err := json.Marshal(log.PromptSignals)
	if err != nil {
		return fmt.Errorf("store: marshal prompt signals: %w", err)
	}

	var errText *string
	if log.Error != "" {
		errText = &log.Error
	}

	_, err = s.pool.Exec(ctx, `
		INSERT INTO menu_idea_agent_calls (
			batch_run_id, menu_item_id, menu_item_name, called_at,
			adjacent_signal_count, promotion_signal_count, prompt_signals, prompt_text, raw_response,
			idea_count, error
		) VALUES (
			$1, $2, $3, $4,
			$5, $6, $7::jsonb, $8, $9,
			$10, $11
		)`,
		batchRunID, log.MenuItemID, log.MenuItemName, log.CalledAt,
		log.AdjacentSignalCount, log.PromotionSignalCount, string(promptSignals), nullIfEmpty(log.PromptText), nullIfEmpty(log.RawResponse),
		log.IdeaCount, errText,
	)
	if err != nil {
		return fmt.Errorf("store: save menu idea call log: %w", err)
	}
	return nil
}

// nullIfEmpty maps an empty string to a nil parameter so the column stores
// SQL NULL rather than an empty string, distinguishing "never reached the
// LLM" from "the LLM returned an empty string".
func nullIfEmpty(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}
