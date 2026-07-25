//go:build integration

package store

import (
	"context"
	"os"
	"testing"
	"time"

	"fbperformance/internal/agents/trend"
)

// testDatabaseURL returns TEST_DATABASE_URL if set, else the local
// docker-compose connection string documented in the README.
func testDatabaseURL() string {
	if v := os.Getenv("TEST_DATABASE_URL"); v != "" {
		return v
	}
	return "postgres://postgres:devpassword@localhost:5440/fbperformance?sslmode=disable"
}

// vectorAlongAxis returns a trend.EmbeddingDimensions-length vector with a
// single 1.0 at the given axis and zeros elsewhere — a simple way to
// construct vectors with known, hand-computable cosine distances from each
// other (e.g. two identical axes have distance 0; two different axes are
// orthogonal, distance 1).
func vectorAlongAxis(axis int) []float32 {
	v := make([]float32, trend.EmbeddingDimensions)
	v[axis] = 1.0
	return v
}

// vectorBlend returns a normalized blend of axis 0 and axis 1 — used to
// construct a vector at a known, intermediate cosine distance from
// vectorAlongAxis(0).
func vectorBlend() []float32 {
	v := make([]float32, trend.EmbeddingDimensions)
	const c = 0.7071068 // 1/sqrt(2), so the vector is already unit-length
	v[0] = c
	v[1] = c
	return v
}

func TestSearchSimilar_OrderAndCutoff(t *testing.T) {
	ctx := context.Background()
	pool, err := Connect(ctx, testDatabaseURL())
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	defer pool.Close()

	signalStore := NewTrendSignalStore(pool)
	now := time.Now().UTC()

	fixtures := []struct {
		externalID string
		embedding  []float32
	}{
		{"integration-test-close", vectorAlongAxis(0)}, // cosine distance 0 from query
		{"integration-test-mid", vectorBlend()},        // cosine distance ~0.293 from query
		{"integration-test-far", vectorAlongAxis(1)},   // cosine distance 1 (orthogonal) from query
	}
	for _, f := range fixtures {
		err := signalStore.Upsert(ctx, trend.Signal{
			Source:     "integration-test",
			ExternalID: f.externalID,
			Caption:    "fixture",
			Hashtags:   []string{"fixture"},
			ViewCount:  1,
			PostedAt:   now,
		}, f.embedding)
		if err != nil {
			t.Fatalf("upsert fixture %s: %v", f.externalID, err)
		}
	}
	t.Cleanup(func() {
		for _, f := range fixtures {
			_, _ = pool.Exec(ctx, `DELETE FROM trend_signals WHERE source = 'integration-test' AND external_id = $1`, f.externalID)
		}
	})

	query := vectorAlongAxis(0)
	results, err := signalStore.SearchSimilar(ctx, query, 10, now.Add(-time.Hour))
	if err != nil {
		t.Fatalf("SearchSimilar: %v", err)
	}

	var got []string
	for _, r := range results {
		if r.Source == "integration-test" {
			got = append(got, r.ExternalID)
		}
	}
	if len(got) != 2 {
		t.Fatalf("expected 2 results within the similarity cutoff (close, mid), got %v", got)
	}
	if got[0] != "integration-test-close" || got[1] != "integration-test-mid" {
		t.Fatalf("expected [close mid] ordered by similarity, got %v", got)
	}
}
