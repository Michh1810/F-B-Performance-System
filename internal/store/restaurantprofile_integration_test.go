//go:build integration

package store

import (
	"context"
	"testing"
)

func TestRestaurantProfileStore_UpsertAndGet(t *testing.T) {
	ctx := context.Background()
	pool, err := Connect(ctx, testDatabaseURL())
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	// Registered before the restore-original Cleanup below so it runs
	// after it (Cleanup is LIFO) — see the identical note in
	// menuideas_integration_test.go.
	t.Cleanup(func() { pool.Close() })

	profileStore := NewRestaurantProfileStore(pool)

	// restaurant_profile is a true singleton (fixed row id), so this test
	// runs against whatever the shared dev DB already has. Capture it and
	// restore it afterward rather than deleting, so the test doesn't
	// clobber a real description set outside this test.
	original, err := profileStore.Get(ctx)
	if err != nil {
		t.Fatalf("Get (original): %v", err)
	}
	t.Cleanup(func() {
		if original.Description == "" {
			_, _ = pool.Exec(ctx, `DELETE FROM restaurant_profile`)
			return
		}
		_, _ = profileStore.Upsert(ctx, original.Description)
	})

	const testDescription = "restaurant-profile-test: Korean-Mexican fusion fast casual"
	updated, err := profileStore.Upsert(ctx, testDescription)
	if err != nil {
		t.Fatalf("Upsert: %v", err)
	}
	if updated.Description != testDescription {
		t.Fatalf("Upsert().Description = %q, want %q", updated.Description, testDescription)
	}
	if updated.UpdatedAt == nil {
		t.Fatalf("expected UpdatedAt to be set after Upsert")
	}

	got, err := profileStore.Get(ctx)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got.Description != testDescription {
		t.Fatalf("Get().Description = %q, want %q", got.Description, testDescription)
	}

	const secondDescription = "restaurant-profile-test: now a ramen and matcha cafe"
	updated2, err := profileStore.Upsert(ctx, secondDescription)
	if err != nil {
		t.Fatalf("Upsert (second): %v", err)
	}
	if updated2.Description != secondDescription {
		t.Fatalf("expected Upsert to overwrite the singleton row rather than create a second one, got %q", updated2.Description)
	}
}
