package handlers

import (
	"encoding/json"
	"net/http"
	"time"

	"fbperformance/internal/services/trendingest"
)

// TrendIngestStatusProvider is the subset of trendingest.Service this
// handler depends on.
type TrendIngestStatusProvider interface {
	Status() trendingest.RunStatus
}

// TrendIngestHandler exposes the background sweep trendingest.Service.Status
// tracks — so the frontend can poll and show a "scraping in progress" state
// after approving a hashtag suggestion (see TrendHashtagsHandler.UpdateStatus).
type TrendIngestHandler struct {
	service TrendIngestStatusProvider
}

func NewTrendIngestHandler(service TrendIngestStatusProvider) *TrendIngestHandler {
	return &TrendIngestHandler{service: service}
}

type trendIngestStatusResponse struct {
	Status         string   `json:"status"`
	Hashtags       []string `json:"hashtags"`
	StartedAt      *string  `json:"started_at"`
	CompletedAt    *string  `json:"completed_at"`
	VideosUpserted int      `json:"videos_upserted"`
	Error          string   `json:"error,omitempty"`
}

// Status handles GET /api/trend-ingest/status.
func (h *TrendIngestHandler) Status(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	status := h.service.Status()

	// Coerce a nil Hashtags (the zero value before any run has ever been
	// triggered) to `[]` rather than letting it serialize as JSON `null` —
	// same reasoning as listIdeas/listHashtagSuggestions on the frontend,
	// applied here instead since this type has no frontend-side wrapper.
	hashtags := status.Hashtags
	if hashtags == nil {
		hashtags = []string{}
	}

	resp := trendIngestStatusResponse{
		Status:         status.Status,
		Hashtags:       hashtags,
		VideosUpserted: status.VideosUpserted,
		Error:          status.Error,
	}
	if status.StartedAt != nil {
		s := status.StartedAt.Format(time.RFC3339)
		resp.StartedAt = &s
	}
	if status.CompletedAt != nil {
		s := status.CompletedAt.Format(time.RFC3339)
		resp.CompletedAt = &s
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(resp)
}
