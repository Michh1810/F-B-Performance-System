package store

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"fbperformance/internal/agents/menuidea"
)

// ErrIdeaNotFound is returned by UpdateStatus when id doesn't match any row.
var ErrIdeaNotFound = errors.New("menu idea not found")

// ErrPriceRequiredToPromote is returned by PromoteToMenuItem when promoting
// a "tweak"-kind idea (a genuinely new item) without a positive price and
// non-negative COGS — required because a human, not the AI, is responsible
// for pricing a real menu item.
var ErrPriceRequiredToPromote = errors.New("price_cents (>0) and cogs_cents (>=0) are required to promote a new-item idea")

// MenuIdeaStore persists and retrieves Menu Idea Agent output for human
// review.
type MenuIdeaStore struct {
	pool *pgxpool.Pool
}

func NewMenuIdeaStore(pool *pgxpool.Pool) *MenuIdeaStore {
	return &MenuIdeaStore{pool: pool}
}

// Save persists one generated idea from a batch run, defaulting its status
// to "new" if unset.
func (s *MenuIdeaStore) Save(ctx context.Context, idea menuidea.StoredIdea) error {
	status := idea.Status
	if status == "" {
		status = "new"
	}
	hashtags, err := json.Marshal(idea.SourceHashtags)
	if err != nil {
		return fmt.Errorf("store: marshal source hashtags: %w", err)
	}
	videoURLs, err := json.Marshal(idea.SourceVideoURLs)
	if err != nil {
		return fmt.Errorf("store: marshal source video urls: %w", err)
	}

	kind := idea.Kind
	if kind == "" {
		kind = "tweak"
	}

	_, err = s.pool.Exec(ctx, `
		INSERT INTO menu_ideas (
			batch_run_id, idea_name, idea_description, suggested_category, rationale,
			inspired_by_menu_item_id, inspired_by_menu_item_name,
			source_hashtags, source_signal_count, source_total_views, source_video_urls, status, kind
		) VALUES (
			$1, $2, $3, $4, $5,
			$6, $7,
			$8::jsonb, $9, $10, $11::jsonb, $12, $13
		)`,
		idea.BatchRunID, idea.Name, idea.Description, idea.SuggestedCategory, idea.Rationale,
		idea.InspiredByMenuItemID, idea.InspiredByMenuItemName,
		string(hashtags), idea.SourceSignalCount, idea.SourceTotalViews, string(videoURLs), status, kind,
	)
	if err != nil {
		return fmt.Errorf("store: save menu idea: %w", err)
	}
	return nil
}

// List returns ideas, most recently generated first, optionally filtered by
// status ("" means no filter).
func (s *MenuIdeaStore) List(ctx context.Context, status string) ([]menuidea.StoredIdea, error) {
	query := `
		SELECT id, batch_run_id, idea_name, idea_description, suggested_category, rationale,
			inspired_by_menu_item_id, inspired_by_menu_item_name,
			source_hashtags, source_signal_count, source_total_views, source_video_urls,
			status, generated_at, reviewed_at, kind, created_menu_item_id
		FROM menu_ideas`

	var rows pgx.Rows
	var err error
	if status != "" {
		rows, err = s.pool.Query(ctx, query+` WHERE status = $1 ORDER BY generated_at DESC`, status)
	} else {
		rows, err = s.pool.Query(ctx, query+` ORDER BY generated_at DESC`)
	}
	if err != nil {
		return nil, fmt.Errorf("store: list menu ideas: %w", err)
	}
	defer rows.Close()

	var ideas []menuidea.StoredIdea
	for rows.Next() {
		var idea menuidea.StoredIdea
		var hashtags []byte
		var videoURLs []byte
		if err := rows.Scan(
			&idea.ID, &idea.BatchRunID, &idea.Name, &idea.Description, &idea.SuggestedCategory, &idea.Rationale,
			&idea.InspiredByMenuItemID, &idea.InspiredByMenuItemName,
			&hashtags, &idea.SourceSignalCount, &idea.SourceTotalViews, &videoURLs,
			&idea.Status, &idea.GeneratedAt, &idea.ReviewedAt, &idea.Kind, &idea.CreatedMenuItemID,
		); err != nil {
			return nil, fmt.Errorf("store: scan menu idea: %w", err)
		}
		if err := json.Unmarshal(hashtags, &idea.SourceHashtags); err != nil {
			return nil, fmt.Errorf("store: unmarshal source hashtags: %w", err)
		}
		if err := json.Unmarshal(videoURLs, &idea.SourceVideoURLs); err != nil {
			return nil, fmt.Errorf("store: unmarshal source video urls: %w", err)
		}
		ideas = append(ideas, idea)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("store: list menu ideas: %w", err)
	}
	return ideas, nil
}

// UpdateStatus transitions an idea's review status and stamps reviewed_at.
// Returns ErrIdeaNotFound if id doesn't match any row. Not used for
// "promoted" — see PromoteToMenuItem, which has side effects UpdateStatus
// deliberately doesn't have.
func (s *MenuIdeaStore) UpdateStatus(ctx context.Context, id uuid.UUID, status string) error {
	tag, err := s.pool.Exec(ctx, `
		UPDATE menu_ideas SET status = $1, reviewed_at = CURRENT_TIMESTAMP WHERE id = $2`,
		status, id,
	)
	if err != nil {
		return fmt.Errorf("store: update menu idea status: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrIdeaNotFound
	}
	return nil
}

// PromoteToMenuItem approves an idea for real use:
//   - "promotion"-kind ideas (feature an existing item) just transition to
//     status "promoted" — the item already exists, nothing to create.
//   - "tweak"-kind ideas (a genuinely new item) additionally create a real
//     menu_items row from the idea's name/category, priced by the human
//     approving it (see ErrPriceRequiredToPromote), and link it via
//     created_menu_item_id.
//
// Idempotent: an idea already promoted (created_menu_item_id set, or
// status already "promoted") is returned as-is without re-running side
// effects or creating a second menu_items row, even if called again with
// different price/cogs. Returns ErrIdeaNotFound if id doesn't match any
// row.
func (s *MenuIdeaStore) PromoteToMenuItem(ctx context.Context, id uuid.UUID, priceCents, cogsCents int64) (menuidea.StoredIdea, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return menuidea.StoredIdea{}, fmt.Errorf("store: begin promote transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	var idea menuidea.StoredIdea
	var hashtags, videoURLs []byte
	err = tx.QueryRow(ctx, `
		SELECT id, batch_run_id, idea_name, idea_description, suggested_category, rationale,
			inspired_by_menu_item_id, inspired_by_menu_item_name,
			source_hashtags, source_signal_count, source_total_views, source_video_urls,
			status, generated_at, reviewed_at, kind, created_menu_item_id
		FROM menu_ideas WHERE id = $1 FOR UPDATE`, id,
	).Scan(
		&idea.ID, &idea.BatchRunID, &idea.Name, &idea.Description, &idea.SuggestedCategory, &idea.Rationale,
		&idea.InspiredByMenuItemID, &idea.InspiredByMenuItemName,
		&hashtags, &idea.SourceSignalCount, &idea.SourceTotalViews, &videoURLs,
		&idea.Status, &idea.GeneratedAt, &idea.ReviewedAt, &idea.Kind, &idea.CreatedMenuItemID,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return menuidea.StoredIdea{}, ErrIdeaNotFound
		}
		return menuidea.StoredIdea{}, fmt.Errorf("store: load idea to promote: %w", err)
	}
	if err := json.Unmarshal(hashtags, &idea.SourceHashtags); err != nil {
		return menuidea.StoredIdea{}, fmt.Errorf("store: unmarshal source hashtags: %w", err)
	}
	if err := json.Unmarshal(videoURLs, &idea.SourceVideoURLs); err != nil {
		return menuidea.StoredIdea{}, fmt.Errorf("store: unmarshal source video urls: %w", err)
	}

	if idea.CreatedMenuItemID != nil || idea.Status == "promoted" {
		if err := tx.Commit(ctx); err != nil {
			return menuidea.StoredIdea{}, fmt.Errorf("store: commit promote transaction: %w", err)
		}
		return idea, nil
	}

	if idea.Kind == "promotion" {
		err = tx.QueryRow(ctx, `
			UPDATE menu_ideas SET status = 'promoted', reviewed_at = CURRENT_TIMESTAMP
			WHERE id = $1 RETURNING status, reviewed_at`, id,
		).Scan(&idea.Status, &idea.ReviewedAt)
		if err != nil {
			return menuidea.StoredIdea{}, fmt.Errorf("store: update promoted status: %w", err)
		}
		if err := tx.Commit(ctx); err != nil {
			return menuidea.StoredIdea{}, fmt.Errorf("store: commit promote transaction: %w", err)
		}
		return idea, nil
	}

	if priceCents <= 0 || cogsCents < 0 {
		return menuidea.StoredIdea{}, ErrPriceRequiredToPromote
	}

	var newItemID uuid.UUID
	err = tx.QueryRow(ctx, `
		INSERT INTO menu_items (name, category, current_price, cogs, is_active)
		VALUES ($1, $2, $3::numeric / 100, $4::numeric / 100, true)
		RETURNING id`,
		idea.Name, idea.SuggestedCategory, priceCents, cogsCents,
	).Scan(&newItemID)
	if err != nil {
		return menuidea.StoredIdea{}, fmt.Errorf("store: create menu item from idea: %w", err)
	}

	err = tx.QueryRow(ctx, `
		UPDATE menu_ideas SET status = 'promoted', reviewed_at = CURRENT_TIMESTAMP, created_menu_item_id = $1
		WHERE id = $2 RETURNING status, reviewed_at, created_menu_item_id`,
		newItemID, id,
	).Scan(&idea.Status, &idea.ReviewedAt, &idea.CreatedMenuItemID)
	if err != nil {
		return menuidea.StoredIdea{}, fmt.Errorf("store: link created menu item: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return menuidea.StoredIdea{}, fmt.Errorf("store: commit promote transaction: %w", err)
	}
	return idea, nil
}
