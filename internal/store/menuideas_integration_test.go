//go:build integration

package store

import (
	"context"
	"errors"
	"math"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"fbperformance/internal/agents/menuidea"
	"fbperformance/internal/agents/trend"
)

// vectorAtCosine returns a trend.EmbeddingDimensions-length unit vector
// whose cosine similarity to vectorAlongAxis(0) is exactly cosine, i.e.
// cosine distance (1 - cosine) from it — a way to construct fixtures at a
// known, hand-computable distance for exercising SearchRelevant's cutoff.
func vectorAtCosine(cosine float64) []float32 {
	v := make([]float32, trend.EmbeddingDimensions)
	v[0] = float32(cosine)
	v[1] = float32(math.Sqrt(1 - cosine*cosine))
	return v
}

// TestSearchRelevant_Band covers the single relevance cutoff: everything
// under relevanceCutoff (0.35) comes back regardless of how close it is —
// including near-duplicates (distance 0) — since promotion-vs-tweak
// classification now happens by caption text (menuidea.mentionsItem), not
// by a second distance band. See relevanceCutoff's doc comment in
// trendsignals.go for why that second band was removed.
func TestSearchRelevant_Band(t *testing.T) {
	ctx := context.Background()
	pool, err := Connect(ctx, testDatabaseURL())
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	// Registered before any fixture-delete Cleanup below so it runs after
	// them (Cleanup is LIFO) — a plain `defer pool.Close()` would run
	// before any t.Cleanup func, since Cleanup fires only after the test
	// function (and its own defers) have already returned, closing the
	// pool out from under the delete and silently no-oping it.
	t.Cleanup(func() { pool.Close() })

	signalStore := NewTrendSignalStore(pool)
	now := time.Now().UTC()

	fixtures := []struct {
		externalID string
		embedding  []float32
	}{
		{"relevant-test-duplicate", vectorAlongAxis(0)},  // distance 0 -- still relevant, included
		{"relevant-test-adjacent", vectorAtCosine(0.8)},  // distance 0.2 -- included
		{"relevant-test-irrelevant", vectorAlongAxis(1)}, // distance 1 (orthogonal) -- excluded
	}
	for _, f := range fixtures {
		err := signalStore.Upsert(ctx, trend.Signal{
			Source:     "relevant-test",
			ExternalID: f.externalID,
			Caption:    "fixture",
			Hashtags:   []string{"fixture"},
			ViewCount:  1,
			PostedAt:   now,
			URL:        "https://www.tiktok.com/@mockuser/video/" + f.externalID,
		}, f.embedding)
		if err != nil {
			t.Fatalf("upsert fixture %s: %v", f.externalID, err)
		}
	}
	t.Cleanup(func() {
		for _, f := range fixtures {
			_, _ = pool.Exec(ctx, `DELETE FROM trend_signals WHERE source = 'relevant-test' AND external_id = $1`, f.externalID)
		}
	})

	query := vectorAlongAxis(0)
	results, err := signalStore.SearchRelevant(ctx, query, 10, now.Add(-time.Hour))
	if err != nil {
		t.Fatalf("SearchRelevant: %v", err)
	}

	var got []string
	for _, r := range results {
		if r.Source == "relevant-test" {
			got = append(got, r.ExternalID)
		}
	}
	if len(got) != 2 {
		t.Fatalf("expected 2 results within the relevance cutoff (duplicate, adjacent), got %v", got)
	}
	if got[0] != "relevant-test-duplicate" || got[1] != "relevant-test-adjacent" {
		t.Fatalf("expected [duplicate adjacent] ordered by similarity, got %v", got)
	}
	for _, r := range results {
		if r.ExternalID == "relevant-test-duplicate" && r.URL != "https://www.tiktok.com/@mockuser/video/relevant-test-duplicate" {
			t.Fatalf("URL = %q, want the fixture's video_url round-tripped through the query", r.URL)
		}
	}
}

func TestMenuItemStore_ListActive(t *testing.T) {
	ctx := context.Background()
	pool, err := Connect(ctx, testDatabaseURL())
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	// Registered before any fixture-delete Cleanup below so it runs after
	// them (Cleanup is LIFO) — a plain `defer pool.Close()` would run
	// before any t.Cleanup func, since Cleanup fires only after the test
	// function (and its own defers) have already returned, closing the
	// pool out from under the delete and silently no-oping it.
	t.Cleanup(func() { pool.Close() })

	var activeID, inactiveID uuid.UUID
	err = pool.QueryRow(ctx, `
		INSERT INTO menu_items (name, category, current_price, cogs, is_active)
		VALUES ('list-active-test-active', 'Test Category', 9.99, 3.00, true) RETURNING id`).Scan(&activeID)
	if err != nil {
		t.Fatalf("insert active fixture: %v", err)
	}
	err = pool.QueryRow(ctx, `
		INSERT INTO menu_items (name, category, current_price, cogs, is_active)
		VALUES ('list-active-test-inactive', 'Test Category', 9.99, 3.00, false) RETURNING id`).Scan(&inactiveID)
	if err != nil {
		t.Fatalf("insert inactive fixture: %v", err)
	}
	t.Cleanup(func() {
		_, _ = pool.Exec(ctx, `DELETE FROM menu_items WHERE id IN ($1, $2)`, activeID, inactiveID)
	})

	items, err := NewMenuItemStore(pool).ListActive(ctx)
	if err != nil {
		t.Fatalf("ListActive: %v", err)
	}

	var sawActive, sawInactive bool
	for _, item := range items {
		if item.ID == activeID {
			sawActive = true
			if item.Name != "list-active-test-active" || item.Category != "Test Category" {
				t.Fatalf("active item = %+v, want matching name/category", item)
			}
		}
		if item.ID == inactiveID {
			sawInactive = true
		}
	}
	if !sawActive {
		t.Fatalf("expected active fixture in ListActive results")
	}
	if sawInactive {
		t.Fatalf("expected inactive fixture excluded from ListActive results")
	}
}

func TestMenuIdeaStore_SaveListUpdateStatus(t *testing.T) {
	ctx := context.Background()
	pool, err := Connect(ctx, testDatabaseURL())
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	// Registered before any fixture-delete Cleanup below so it runs after
	// them (Cleanup is LIFO) — a plain `defer pool.Close()` would run
	// before any t.Cleanup func, since Cleanup fires only after the test
	// function (and its own defers) have already returned, closing the
	// pool out from under the delete and silently no-oping it.
	t.Cleanup(func() { pool.Close() })

	var menuItemID uuid.UUID
	err = pool.QueryRow(ctx, `
		INSERT INTO menu_items (name, category, current_price, cogs, is_active)
		VALUES ('idea-store-test-item', 'Test Category', 9.99, 3.00, true) RETURNING id`).Scan(&menuItemID)
	if err != nil {
		t.Fatalf("insert menu item fixture: %v", err)
	}
	t.Cleanup(func() {
		_, _ = pool.Exec(ctx, `DELETE FROM menu_items WHERE id = $1`, menuItemID)
	})

	ideaStore := NewMenuIdeaStore(pool)
	batchRunID := uuid.New()
	idea := menuidea.StoredIdea{
		BatchRunID:             batchRunID,
		InspiredByMenuItemID:   menuItemID,
		InspiredByMenuItemName: "idea-store-test-item",
		Status:                 "new",
		IdeaCandidate: menuidea.IdeaCandidate{
			Name:              "idea-store-test-idea",
			Description:       "a test idea",
			SuggestedCategory: "Test Category",
			Rationale:         "because it's a test",
			SourceHashtags:    []string{"foodtiktok", "birria"},
			SourceSignalCount: 3,
			SourceTotalViews:  5000,
			SourceVideoURLs:   []string{"https://www.tiktok.com/@mockuser/video/idea-store-test-1"},
		},
	}
	if err := ideaStore.Save(ctx, idea); err != nil {
		t.Fatalf("Save: %v", err)
	}
	t.Cleanup(func() {
		_, _ = pool.Exec(ctx, `DELETE FROM menu_ideas WHERE batch_run_id = $1`, batchRunID)
	})

	all, err := ideaStore.List(ctx, "")
	if err != nil {
		t.Fatalf("List(\"\"): %v", err)
	}
	var saved *menuidea.StoredIdea
	for i := range all {
		if all[i].BatchRunID == batchRunID {
			saved = &all[i]
		}
	}
	if saved == nil {
		t.Fatalf("expected saved idea to appear in List(\"\")")
	}
	if saved.Name != "idea-store-test-idea" || saved.Status != "new" {
		t.Fatalf("saved idea = %+v, want name/status matching insert", saved)
	}
	if len(saved.SourceHashtags) != 2 {
		t.Fatalf("SourceHashtags = %v, want 2 entries round-tripped through JSONB", saved.SourceHashtags)
	}
	if len(saved.SourceVideoURLs) != 1 || saved.SourceVideoURLs[0] != "https://www.tiktok.com/@mockuser/video/idea-store-test-1" {
		t.Fatalf("SourceVideoURLs = %v, want the one evidence clip URL round-tripped through JSONB", saved.SourceVideoURLs)
	}

	filtered, err := ideaStore.List(ctx, "new")
	if err != nil {
		t.Fatalf("List(\"new\"): %v", err)
	}
	found := false
	for _, i := range filtered {
		if i.ID == saved.ID {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected saved idea to appear in List(\"new\")")
	}

	if err := ideaStore.UpdateStatus(ctx, saved.ID, "promoted"); err != nil {
		t.Fatalf("UpdateStatus: %v", err)
	}

	afterUpdate, err := ideaStore.List(ctx, "promoted")
	if err != nil {
		t.Fatalf("List(\"promoted\"): %v", err)
	}
	found = false
	for _, i := range afterUpdate {
		if i.ID == saved.ID {
			found = true
			if i.ReviewedAt == nil {
				t.Fatalf("expected ReviewedAt to be stamped after UpdateStatus")
			}
		}
	}
	if !found {
		t.Fatalf("expected promoted idea to appear in List(\"promoted\") after UpdateStatus")
	}
}

func TestMenuIdeaStore_UpdateStatus_NotFound(t *testing.T) {
	ctx := context.Background()
	pool, err := Connect(ctx, testDatabaseURL())
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	// Registered before any fixture-delete Cleanup below so it runs after
	// them (Cleanup is LIFO) — a plain `defer pool.Close()` would run
	// before any t.Cleanup func, since Cleanup fires only after the test
	// function (and its own defers) have already returned, closing the
	// pool out from under the delete and silently no-oping it.
	t.Cleanup(func() { pool.Close() })

	err = NewMenuIdeaStore(pool).UpdateStatus(ctx, uuid.New(), "reviewed")
	if !errors.Is(err, ErrIdeaNotFound) {
		t.Fatalf("UpdateStatus error = %v, want ErrIdeaNotFound", err)
	}
}

// insertTestIdeaFixture inserts a menu_ideas row directly (bypassing Save,
// since Save always defaults status to "new" and we need to control kind
// precisely), returning its id. Registers cleanup for both the idea and any
// menu_items row PromoteToMenuItem might create from it.
func insertTestIdeaFixture(t *testing.T, ctx context.Context, pool *pgxpool.Pool, name, kind string) uuid.UUID {
	t.Helper()
	var id uuid.UUID
	err := pool.QueryRow(ctx, `
		INSERT INTO menu_ideas (
			batch_run_id, idea_name, idea_description, suggested_category, rationale,
			inspired_by_menu_item_name, source_hashtags, source_video_urls, status, kind
		) VALUES (
			$1, $2, 'a test idea', 'Test Category', 'because it is a test',
			'promote-test-source-item', '[]'::jsonb, '[]'::jsonb, 'new', $3
		) RETURNING id`,
		uuid.New(), name, kind,
	).Scan(&id)
	if err != nil {
		t.Fatalf("insert idea fixture %q: %v", name, err)
	}
	t.Cleanup(func() {
		_, _ = pool.Exec(ctx, `DELETE FROM menu_ideas WHERE id = $1`, id)
		_, _ = pool.Exec(ctx, `DELETE FROM menu_items WHERE name = $1`, name)
	})
	return id
}

func TestMenuIdeaStore_PromoteToMenuItem_TweakCreatesRealMenuItem(t *testing.T) {
	ctx := context.Background()
	pool, err := Connect(ctx, testDatabaseURL())
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	t.Cleanup(func() { pool.Close() })

	ideaStore := NewMenuIdeaStore(pool)
	ideaID := insertTestIdeaFixture(t, ctx, pool, "promote-test-tweak-item", "tweak")

	promoted, err := ideaStore.PromoteToMenuItem(ctx, ideaID, 1200, 400)
	if err != nil {
		t.Fatalf("PromoteToMenuItem: %v", err)
	}
	if promoted.Status != "promoted" {
		t.Fatalf("Status = %q, want promoted", promoted.Status)
	}
	if promoted.CreatedMenuItemID == nil {
		t.Fatalf("expected CreatedMenuItemID to be set for a tweak-kind promotion")
	}

	var name, category string
	var price, cogs float64
	var isActive bool
	err = pool.QueryRow(ctx, `SELECT name, category, current_price, cogs, is_active FROM menu_items WHERE id = $1`, *promoted.CreatedMenuItemID).
		Scan(&name, &category, &price, &cogs, &isActive)
	if err != nil {
		t.Fatalf("expected a real menu_items row to have been created: %v", err)
	}
	if name != "promote-test-tweak-item" || category != "Test Category" || !isActive {
		t.Fatalf("created menu_items row = (name=%q, category=%q, active=%v), want matching the idea", name, category, isActive)
	}
	if price != 12.00 || cogs != 4.00 {
		t.Fatalf("created menu_items row = (price=%.2f, cogs=%.2f), want (12.00, 4.00)", price, cogs)
	}

	// Idempotent: promoting again (even with different price/cogs) must not
	// create a second menu_items row.
	promotedAgain, err := ideaStore.PromoteToMenuItem(ctx, ideaID, 9999, 9999)
	if err != nil {
		t.Fatalf("PromoteToMenuItem (second call): %v", err)
	}
	if *promotedAgain.CreatedMenuItemID != *promoted.CreatedMenuItemID {
		t.Fatalf("expected the second PromoteToMenuItem call to return the same created_menu_item_id, got %v vs %v", promotedAgain.CreatedMenuItemID, promoted.CreatedMenuItemID)
	}
	var menuItemCount int
	err = pool.QueryRow(ctx, `SELECT count(*) FROM menu_items WHERE name = 'promote-test-tweak-item'`).Scan(&menuItemCount)
	if err != nil {
		t.Fatalf("count menu_items: %v", err)
	}
	if menuItemCount != 1 {
		t.Fatalf("expected exactly 1 menu_items row after promoting twice, got %d", menuItemCount)
	}
}

func TestMenuIdeaStore_PromoteToMenuItem_TweakWithoutPriceIsRejected(t *testing.T) {
	ctx := context.Background()
	pool, err := Connect(ctx, testDatabaseURL())
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	t.Cleanup(func() { pool.Close() })

	ideaStore := NewMenuIdeaStore(pool)
	ideaID := insertTestIdeaFixture(t, ctx, pool, "promote-test-no-price-item", "tweak")

	_, err = ideaStore.PromoteToMenuItem(ctx, ideaID, 0, 0)
	if !errors.Is(err, ErrPriceRequiredToPromote) {
		t.Fatalf("PromoteToMenuItem error = %v, want ErrPriceRequiredToPromote", err)
	}

	var menuItemCount int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM menu_items WHERE name = 'promote-test-no-price-item'`).Scan(&menuItemCount); err != nil {
		t.Fatalf("count menu_items: %v", err)
	}
	if menuItemCount != 0 {
		t.Fatalf("expected no menu_items row created when price is missing, got %d", menuItemCount)
	}
}

func TestMenuIdeaStore_PromoteToMenuItem_PromotionKindCreatesNoMenuItem(t *testing.T) {
	ctx := context.Background()
	pool, err := Connect(ctx, testDatabaseURL())
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	t.Cleanup(func() { pool.Close() })

	ideaStore := NewMenuIdeaStore(pool)
	ideaID := insertTestIdeaFixture(t, ctx, pool, "promote-test-promotion-item", "promotion")

	promoted, err := ideaStore.PromoteToMenuItem(ctx, ideaID, 0, 0)
	if err != nil {
		t.Fatalf("PromoteToMenuItem: %v", err)
	}
	if promoted.Status != "promoted" {
		t.Fatalf("Status = %q, want promoted", promoted.Status)
	}
	if promoted.CreatedMenuItemID != nil {
		t.Fatalf("expected CreatedMenuItemID to stay nil for a promotion-kind idea, got %v", promoted.CreatedMenuItemID)
	}

	var menuItemCount int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM menu_items WHERE name = 'promote-test-promotion-item'`).Scan(&menuItemCount); err != nil {
		t.Fatalf("count menu_items: %v", err)
	}
	if menuItemCount != 0 {
		t.Fatalf("expected no menu_items row created for a promotion-kind idea, got %d", menuItemCount)
	}
}

func TestMenuIdeaStore_PromoteToMenuItem_NotFound(t *testing.T) {
	ctx := context.Background()
	pool, err := Connect(ctx, testDatabaseURL())
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	t.Cleanup(func() { pool.Close() })

	_, err = NewMenuIdeaStore(pool).PromoteToMenuItem(ctx, uuid.New(), 1000, 300)
	if !errors.Is(err, ErrIdeaNotFound) {
		t.Fatalf("PromoteToMenuItem error = %v, want ErrIdeaNotFound", err)
	}
}
