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

	// Comments is up to commentsPerVideo top comments on this video,
	// fetched via a second call to the actor's (shared, per-run) comments
	// dataset — see Client.attachComments. Empty if the actor returned no
	// commentsDatasetUrl, or if that follow-up fetch failed; comments are
	// best-effort enrichment, not required for FetchByHashtags to succeed.
	Comments []Comment
}

// Comment is a single TikTok comment on a video.
type Comment struct {
	ID             string
	Text           string
	DiggCount      int64
	AuthorUsername string
	PostedAt       time.Time
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

	// CommentsDatasetURL points to a *separate* Apify dataset holding
	// comments for every video in this actor run (not just this one),
	// distinguished per-comment by videoWebUrl — confirmed against a live
	// run: every item in one run shares the identical URL. See
	// Client.attachComments, which fetches it once per FetchByHashtags
	// call and groups results back onto each Video by URL.
	CommentsDatasetURL string `json:"commentsDatasetUrl"`
}

// toVideo converts a raw Apify dataset item into the service's Video DTO.
// A malformed/missing createTimeISO is not treated as fatal; PostedAt is
// simply left zero-valued. Comments is left empty here — attachComments
// fills it in as a second pass, since comments live in a separate dataset.
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

// apifyCommentItem is the raw shape of one item in the actor's comments
// dataset. Field names verified against a live actor run with
// commentsPerPost set — this is not documented on the actor's public store
// page, which only lists output field names for videos.
type apifyCommentItem struct {
	VideoWebURL   string `json:"videoWebUrl"`
	Cid           string `json:"cid"`
	Text          string `json:"text"`
	DiggCount     int64  `json:"diggCount"`
	UniqueID      string `json:"uniqueId"`
	CreateTimeISO string `json:"createTimeISO"`
}

// toComment converts a raw comment dataset item into the service's Comment
// DTO. Like toVideo, a malformed/missing createTimeISO is not fatal.
func (c apifyCommentItem) toComment() Comment {
	postedAt, _ := time.Parse(time.RFC3339, c.CreateTimeISO)
	return Comment{
		ID:             c.Cid,
		Text:           c.Text,
		DiggCount:      c.DiggCount,
		AuthorUsername: c.UniqueID,
		PostedAt:       postedAt,
	}
}
