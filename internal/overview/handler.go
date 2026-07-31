package overview

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

type Handler struct {
	s *Service
}

func NewHandler(s *Service) *Handler {
	return &Handler{s: s}
}

// HandleGetOverview returns the overview dashboard data for the frontend
func (h *Handler) HandleGetOverview(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	// For now, default to the last 30 days vs previous 30 days
	to := time.Now()
	from := to.AddDate(0, 0, -30)

	// If query params are provided, parse them
	if fromStr := r.URL.Query().Get("from"); fromStr != "" {
		if t, err := time.Parse(time.RFC3339, fromStr); err == nil {
			from = t
		}
	}
	if toStr := r.URL.Query().Get("to"); toStr != "" {
		if t, err := time.Parse(time.RFC3339, toStr); err == nil {
			to = t
		}
	}

	data, err := h.s.GetOverviewData(r.Context(), from, to)
	if err != nil {
		fmt.Printf("HandleGetOverview Error: %v\n", err)
		http.Error(w, "failed to load overview data", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(data); err != nil {
		fmt.Printf("HandleGetOverview Encoding Error: %v\n", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}
