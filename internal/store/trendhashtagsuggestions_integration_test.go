//go:build integration

package store

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
)

func TestTrendHashtagSuggestionStore_SaveListUpdateStatusLatestApproved(t *testing.T) {
	ctx := context.Background()
	pool, err := Connect(ctx, testDatabaseURL())
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	// Registered before the fixture-delete Cleanup below so it runs after
	// it (Cleanup is LIFO) — see the identical note in
	// menuideas_integration_test.go.
	t.Cleanup(func() { pool.Close() })

	suggestionStore := NewTrendHashtagSuggestionStore(pool)

	const marker = "hashtag-suggestion-test-marker"
	saved, err := suggestionStore.Save(ctx, "hashtag-suggestion-test description", []string{marker, "foodtiktok"})
	if err != nil {
		t.Fatalf("Save: %v", err)
	}
	t.Cleanup(func() {
		_, _ = pool.Exec(ctx, `DELETE FROM trend_hashtag_suggestions WHERE id = $1`, saved.ID)
	})

	if saved.Status != "pending" {
		t.Fatalf("Status = %q, want pending on save", saved.Status)
	}
	if len(saved.Hashtags) != 2 {
		t.Fatalf("Hashtags = %v, want 2 entries", saved.Hashtags)
	}

	pending, err := suggestionStore.List(ctx, "pending")
	if err != nil {
		t.Fatalf(`List("pending"): %v`, err)
	}
	found := false
	for _, s := range pending {
		if s.ID == saved.ID {
			found = true
		}
	}
	if !found {
		t.Fatalf(`expected saved suggestion to appear in List("pending")`)
	}

	if hasMarker(t, ctx, suggestionStore, marker) {
		t.Fatalf("expected LatestApproved not to include a still-pending suggestion's hashtags")
	}

	if err := suggestionStore.UpdateStatus(ctx, saved.ID, "approved"); err != nil {
		t.Fatalf("UpdateStatus: %v", err)
	}

	if !hasMarker(t, ctx, suggestionStore, marker) {
		t.Fatalf("expected LatestApproved to return the just-approved suggestion's hashtags")
	}

	approvedList, err := suggestionStore.List(ctx, "approved")
	if err != nil {
		t.Fatalf(`List("approved"): %v`, err)
	}
	found = false
	for _, s := range approvedList {
		if s.ID == saved.ID {
			found = true
			if s.ReviewedAt == nil {
				t.Fatalf("expected ReviewedAt to be stamped after UpdateStatus")
			}
		}
	}
	if !found {
		t.Fatalf(`expected approved suggestion to appear in List("approved")`)
	}
}

// hasMarker reports whether LatestApproved's current result includes
// marker, tolerating that other real approved suggestions may also exist
// in this shared dev DB.
func hasMarker(t *testing.T, ctx context.Context, s *TrendHashtagSuggestionStore, marker string) bool {
	t.Helper()
	hashtags, err := s.LatestApproved(ctx)
	if err != nil {
		t.Fatalf("LatestApproved: %v", err)
	}
	for _, h := range hashtags {
		if h == marker {
			return true
		}
	}
	return false
}

func TestTrendHashtagSuggestionStore_UpdateStatus_NotFound(t *testing.T) {
	ctx := context.Background()
	pool, err := Connect(ctx, testDatabaseURL())
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	t.Cleanup(func() { pool.Close() })

	err = NewTrendHashtagSuggestionStore(pool).UpdateStatus(ctx, uuid.New(), "approved")
	if !errors.Is(err, ErrHashtagSuggestionNotFound) {
		t.Fatalf("UpdateStatus error = %v, want ErrHashtagSuggestionNotFound", err)
	}
}
