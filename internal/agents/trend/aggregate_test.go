package trend

import "testing"

func TestAggregateSignals(t *testing.T) {
	signals := []Signal{
		{ExternalID: "1", Caption: "smash burger", ViewCount: 100000, LikeCount: 8000, CommentCount: 200, ShareCount: 300, Hashtags: []string{"smashburger", "foodtiktok"}},
		{ExternalID: "2", Caption: "another smash burger", ViewCount: 50000, LikeCount: 4000, CommentCount: 100, ShareCount: 150, Hashtags: []string{"smashburger"}},
	}

	snap := aggregateSignals(signals)

	if snap.VideoCount != 2 {
		t.Fatalf("VideoCount = %d, want 2", snap.VideoCount)
	}
	if snap.TotalViews != 150000 {
		t.Fatalf("TotalViews = %d, want 150000", snap.TotalViews)
	}
	if snap.TotalLikes != 12000 || snap.TotalComments != 300 || snap.TotalShares != 450 {
		t.Fatalf("aggregates = likes:%d comments:%d shares:%d, want 12000/300/450", snap.TotalLikes, snap.TotalComments, snap.TotalShares)
	}
	wantEngagement := float64(12000+300+450) / float64(150000)
	if diff := snap.EngagementRate - wantEngagement; diff > 1e-9 || diff < -1e-9 {
		t.Fatalf("EngagementRate = %v, want %v", snap.EngagementRate, wantEngagement)
	}
	if len(snap.TopHashtags) == 0 || snap.TopHashtags[0] != "smashburger" {
		t.Fatalf("TopHashtags = %v, want first entry 'smashburger' (appears in both videos)", snap.TopHashtags)
	}
}

func TestAggregateSignals_SkipsEmptyHashtagNames(t *testing.T) {
	signals := []Signal{
		{ExternalID: "1", ViewCount: 100, Hashtags: []string{"", "realhashtag"}},
	}
	snap := aggregateSignals(signals)
	for _, h := range snap.TopHashtags {
		if h == "" {
			t.Fatalf("TopHashtags contains an empty string: %v", snap.TopHashtags)
		}
	}
	if len(snap.TopHashtags) != 1 || snap.TopHashtags[0] != "realhashtag" {
		t.Fatalf("TopHashtags = %v, want [realhashtag]", snap.TopHashtags)
	}
}

func TestAggregateSignals_Empty(t *testing.T) {
	snap := aggregateSignals(nil)
	if snap.VideoCount != 0 || snap.TotalViews != 0 || snap.EngagementRate != 0 {
		t.Fatalf("expected zeroed metrics, got %+v", snap)
	}
}

func TestSelectTopSignals(t *testing.T) {
	signals := []Signal{
		{ExternalID: "low", ViewCount: 100},
		{ExternalID: "high", ViewCount: 900},
		{ExternalID: "mid", ViewCount: 500},
	}
	top := selectTopSignals(signals, 2)
	if len(top) != 2 {
		t.Fatalf("expected 2 signals, got %d", len(top))
	}
	if top[0].ExternalID != "high" || top[1].ExternalID != "mid" {
		t.Fatalf("expected [high mid] sorted by ViewCount desc, got [%s %s]", top[0].ExternalID, top[1].ExternalID)
	}
	// original slice order must be unaffected
	if signals[0].ExternalID != "low" || signals[1].ExternalID != "high" || signals[2].ExternalID != "mid" {
		t.Fatalf("selectTopSignals mutated caller's slice order: %+v", signals)
	}
}
