package tiktok

import "time"

// Video is a single TikTok video matched by a hashtag search, mapped from
// the Apify actor's raw output into the fields this service cares about.
type Video struct {
	ID             string
	Caption        string
	AuthorUsername string
	URL            string
	PostedAt       time.Time

	PlayCount    int64
	DiggCount    int64
	CommentCount int64
	ShareCount   int64
	CollectCount int64

	Hashtags []string
}

// apifyHashtag is a single entry in the actor's `hashtags` array.
type apifyHashtag struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Title string `json:"title"`
}

// apifyAuthorMeta is the subset of `authorMeta` this service uses.
type apifyAuthorMeta struct {
	Name string `json:"name"`
}

// apifyVideoItem is the raw shape of one dataset item returned by the
// clockworks/tiktok-scraper Apify actor, restricted to the fields this
// service maps into Video. Field names are verified against the actor's
// live OpenAPI spec; if a field name ever drifts, toVideo is the only place
// that needs to change.
type apifyVideoItem struct {
	ID            string          `json:"id"`
	Text          string          `json:"text"`
	CreateTimeISO string          `json:"createTimeISO"`
	WebVideoURL   string          `json:"webVideoUrl"`
	AuthorMeta    apifyAuthorMeta `json:"authorMeta"`

	PlayCount    int64 `json:"playCount"`
	DiggCount    int64 `json:"diggCount"`
	CommentCount int64 `json:"commentCount"`
	ShareCount   int64 `json:"shareCount"`
	CollectCount int64 `json:"collectCount"`

	Hashtags []apifyHashtag `json:"hashtags"`
}

// toVideo converts a raw Apify dataset item into the service's Video DTO.
// A malformed/missing createTimeISO is not treated as fatal; PostedAt is
// simply left zero-valued.
func (v apifyVideoItem) toVideo() Video {
	hashtags := make([]string, 0, len(v.Hashtags))
	for _, h := range v.Hashtags {
		hashtags = append(hashtags, h.Name)
	}

	postedAt, _ := time.Parse(time.RFC3339, v.CreateTimeISO)

	return Video{
		ID:             v.ID,
		Caption:        v.Text,
		AuthorUsername: v.AuthorMeta.Name,
		URL:            v.WebVideoURL,
		PostedAt:       postedAt,
		PlayCount:      v.PlayCount,
		DiggCount:      v.DiggCount,
		CommentCount:   v.CommentCount,
		ShareCount:     v.ShareCount,
		CollectCount:   v.CollectCount,
		Hashtags:       hashtags,
	}
}
