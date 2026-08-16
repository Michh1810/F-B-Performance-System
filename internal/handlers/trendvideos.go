package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"fbperformance/internal/agents/trend"
)

const (
	defaultTrendVideosLimit = 50
	maxTrendVideosLimit     = 200
)

// TrendVideoLister is the subset of trend-signal storage this handler
// depends on. *store.TrendSignalStore satisfies this as-is.
type TrendVideoLister interface {
	List(ctx context.Context, limit, offset int) (signals []trend.Signal, total int, latestIngestedAt *time.Time, err error)
}

// TrendVideosHandler exposes the raw TikTok trend-signal corpus (the
// scraped/mock videos cmd/trend-ingest and cmd/seed-idea-trends write into
// trend_signals) as a browsable list, independent of the semantic-search
// agents that consume the same table.
type TrendVideosHandler struct {
	signals TrendVideoLister
}

func NewTrendVideosHandler(signals TrendVideoLister) *TrendVideosHandler {
	return &TrendVideosHandler{signals: signals}
}

type trendVideoResponse struct {
	ID           string   `json:"id"`
	Source       string   `json:"source"`
	Caption      string   `json:"caption"`
	Hashtags     []string `json:"hashtags"`
	URL          string   `json:"url"`
	ViewCount    int64    `json:"view_count"`
	LikeCount    int64    `json:"like_count"`
	CommentCount int64    `json:"comment_count"`
	ShareCount   int64    `json:"share_count"`
	PostedAt     *string  `json:"posted_at"`
	IngestedAt   string   `json:"ingested_at"`
}

type listTrendVideosResponse struct {
	Videos []trendVideoResponse `json:"videos"`
	Total  int                  `json:"total"`
	Limit  int                  `json:"limit"`
	Offset int                  `json:"offset"`
	// LastUpdated is the most recent ingested_at across the whole corpus
	// (not just this page) — the "latest change time" banner at the top of
	// the scraped-videos view. Nil when the corpus is empty.
	LastUpdated *string `json:"last_updated"`
}

// List handles GET /api/v1/trend-videos, optionally paginated with
// ?limit=&offset=. Results are ordered most-recently-ingested first, so
// videos[0].ingested_at (when present) doubles as LastUpdated.
func (h *TrendVideosHandler) List(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	limit := defaultTrendVideosLimit
	if v := r.URL.Query().Get("limit"); v != "" {
		parsed, err := strconv.Atoi(v)
		if err != nil || parsed <= 0 {
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(map[string]string{"error": "limit must be a positive integer"})
			return
		}
		if parsed > maxTrendVideosLimit {
			parsed = maxTrendVideosLimit
		}
		limit = parsed
	}

	offset := 0
	if v := r.URL.Query().Get("offset"); v != "" {
		parsed, err := strconv.Atoi(v)
		if err != nil || parsed < 0 {
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(map[string]string{"error": "offset must be a non-negative integer"})
			return
		}
		offset = parsed
	}

	signals, total, latestIngestedAt, err := h.signals.List(r.Context(), limit, offset)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	videos := make([]trendVideoResponse, len(signals))
	for i, sig := range signals {
		var postedAt *string
		if !sig.PostedAt.IsZero() {
			s := sig.PostedAt.Format(time.RFC3339)
			postedAt = &s
		}
		videos[i] = trendVideoResponse{
			ID:           sig.ID.String(),
			Source:       sig.Source,
			Caption:      sig.Caption,
			Hashtags:     sig.Hashtags,
			URL:          sig.URL,
			ViewCount:    sig.ViewCount,
			LikeCount:    sig.LikeCount,
			CommentCount: sig.CommentCount,
			ShareCount:   sig.ShareCount,
			PostedAt:     postedAt,
			IngestedAt:   sig.IngestedAt.Format(time.RFC3339),
		}
	}

	var lastUpdated *string
	if latestIngestedAt != nil {
		s := latestIngestedAt.Format(time.RFC3339)
		lastUpdated = &s
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(listTrendVideosResponse{
		Videos:      videos,
		Total:       total,
		Limit:       limit,
		Offset:      offset,
		LastUpdated: lastUpdated,
	})
}
