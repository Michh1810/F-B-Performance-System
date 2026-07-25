package tiktok

import (
	"encoding/json"
	"testing"
	"time"
)

// fixture is a real (anonymized) shape of a clockworks/tiktok-scraper
// dataset item, per the actor's live OpenAPI spec.
const fixture = `[
	{
		"id": "7123456789012345678",
		"text": "smash burger with garlic aioli #smashburger #foodtiktok",
		"createTimeISO": "2026-07-18T14:30:00.000Z",
		"webVideoUrl": "https://www.tiktok.com/@chefjordan/video/7123456789012345678",
		"authorMeta": {"id": "111", "name": "chefjordan", "nickName": "Chef Jordan", "fans": 50000, "verified": false},
		"playCount": 250000,
		"diggCount": 18000,
		"commentCount": 420,
		"shareCount": 900,
		"collectCount": 1200,
		"hashtags": [
			{"id": "1", "name": "smashburger", "title": "smashburger"},
			{"id": "2", "name": "foodtiktok", "title": "foodtiktok"}
		],
		"isAd": false,
		"isPinned": false
	}
]`

func TestToVideo(t *testing.T) {
	var items []apifyVideoItem
	if err := json.Unmarshal([]byte(fixture), &items); err != nil {
		t.Fatalf("unmarshal fixture: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("expected 1 item, got %d", len(items))
	}

	got := items[0].toVideo()

	wantPostedAt, err := time.Parse(time.RFC3339, "2026-07-18T14:30:00.000Z")
	if err != nil {
		t.Fatalf("parse expected time: %v", err)
	}

	want := Video{
		ID:             "7123456789012345678",
		Caption:        "smash burger with garlic aioli #smashburger #foodtiktok",
		AuthorUsername: "chefjordan",
		URL:            "https://www.tiktok.com/@chefjordan/video/7123456789012345678",
		PostedAt:       wantPostedAt,
		PlayCount:      250000,
		DiggCount:      18000,
		CommentCount:   420,
		ShareCount:     900,
		CollectCount:   1200,
		Hashtags:       []string{"smashburger", "foodtiktok"},
	}

	if got.ID != want.ID || got.Caption != want.Caption || got.AuthorUsername != want.AuthorUsername ||
		got.URL != want.URL || !got.PostedAt.Equal(want.PostedAt) ||
		got.PlayCount != want.PlayCount || got.DiggCount != want.DiggCount ||
		got.CommentCount != want.CommentCount || got.ShareCount != want.ShareCount ||
		got.CollectCount != want.CollectCount || len(got.Hashtags) != len(want.Hashtags) {
		t.Fatalf("toVideo() = %+v, want %+v", got, want)
	}
	for i := range want.Hashtags {
		if got.Hashtags[i] != want.Hashtags[i] {
			t.Fatalf("Hashtags[%d] = %q, want %q", i, got.Hashtags[i], want.Hashtags[i])
		}
	}
}

func TestToVideo_MissingCreateTime(t *testing.T) {
	item := apifyVideoItem{ID: "1", Text: "no timestamp"}
	got := item.toVideo()
	if !got.PostedAt.IsZero() {
		t.Fatalf("expected zero PostedAt for missing createTimeISO, got %v", got.PostedAt)
	}
}
