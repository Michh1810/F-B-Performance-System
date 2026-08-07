package financial

import (
	"math"
	"testing"
)

func TestHasSufficientHistory(t *testing.T) {
	tests := []struct {
		name    string
		summary historySummary
		want    bool
	}{
		{name: "sufficient", summary: historySummary{NonzeroSalesDays: 3, TotalUnits: 10}, want: true},
		{name: "insufficient days", summary: historySummary{NonzeroSalesDays: 2, TotalUnits: 10}, want: false},
		{name: "zero units", summary: historySummary{NonzeroSalesDays: 4, TotalUnits: 0}, want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := hasSufficientHistory(tt.summary); got != tt.want {
				t.Fatalf("hasSufficientHistory() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestRelativeSimilarityIsBounded(t *testing.T) {
	if got := relativeSimilarity(1000, 1000); math.Abs(got-1.0) > 1e-9 {
		t.Fatalf("exact match similarity = %f, want 1.0", got)
	}

	if got := relativeSimilarity(1000, 5000); got < (1.0/3.0)-1e-9 || got > (1.0/3.0)+1e-9 {
		t.Fatalf("clamped high difference similarity = %f, want ~0.3333", got)
	}

	if got := relativeSimilarity(0, 0); math.Abs(got-1.0) > 1e-9 {
		t.Fatalf("zero guard similarity = %f, want 1.0", got)
	}
}

func TestCombinedSimilarityScoreStageBCategoryPenalty(t *testing.T) {
	candidate := menuItemMeta{Category: "Beverage"}
	sameCategory := menuItemMeta{Category: "Beverage", PriceCents: 500, EstimatedCOGSCents: 200}
	differentCategory := menuItemMeta{Category: "Dessert", PriceCents: 500, EstimatedCOGSCents: 200}

	sameScore := combinedSimilarityScore(500, 200, candidate, sameCategory, false)
	differentScore := combinedSimilarityScore(500, 200, candidate, differentCategory, false)
	if !(sameScore > differentScore) {
		t.Fatalf("same-category score (%f) should be greater than cross-category score (%f)", sameScore, differentScore)
	}
}

func TestNormalizeWeightsAndWeightedBaseline(t *testing.T) {
	comparables := []comparableCandidate{
		{Score: 0.9, Baseline: 20},
		{Score: 0.75, Baseline: 15},
		{Score: 0.6, Baseline: 10},
	}

	normalizeWeights(comparables)

	totalWeight := 0.0
	for _, c := range comparables {
		totalWeight += c.Weight
	}
	if math.Abs(totalWeight-1.0) > 1e-9 {
		t.Fatalf("sum(weights) = %f, want 1.0", totalWeight)
	}

	got := weightedBaseline(comparables)
	want := 20*(0.9/2.25) + 15*(0.75/2.25) + 10*(0.6/2.25)
	if math.Abs(got-want) > 1e-9 {
		t.Fatalf("weighted baseline = %f, want %f", got, want)
	}
}

func TestSortComparablesDeterministicTieBreak(t *testing.T) {
	in := []comparableCandidate{
		{menuItemMeta: menuItemMeta{ID: "b"}, Score: 0.9, NonzeroSalesDays: 10, Baseline: 12},
		{menuItemMeta: menuItemMeta{ID: "a"}, Score: 0.9, NonzeroSalesDays: 10, Baseline: 12},
	}
	sortComparables(in)
	if in[0].ID != "a" {
		t.Fatalf("expected deterministic ID tie-break ordering, got first ID=%s", in[0].ID)
	}
}
