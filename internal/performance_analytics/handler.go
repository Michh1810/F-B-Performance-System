package performance_analytics

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"
)

type Handler struct {
	s *Service
}

func NewHandler(s *Service) *Handler {
	return &Handler{s: s}
}

func (h *Handler) HandleSummary(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	from, to, err := parseDateRange(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	data, err := h.s.GetDashboardData(r.Context(), from, to)
	if err != nil {
		http.Error(w, "failed to load summary", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func (h *Handler) HandleMenuItems(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	from, to, err := parseDateRange(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	sortBy := strings.TrimSpace(r.URL.Query().Get("sortBy"))
	if sortBy != "" && sortBy != "revenue" && sortBy != "unitsSold" && sortBy != "profitMargin" {
		http.Error(w, "invalid sortBy", http.StatusBadRequest)
		return
	}
	performanceCategory := strings.ToLower(strings.TrimSpace(r.URL.Query().Get("performanceCategory")))
	if performanceCategory != "" &&
		performanceCategory != "star" &&
		performanceCategory != "plowhorse" &&
		performanceCategory != "puzzle" &&
		performanceCategory != "dog" {
		http.Error(w, "invalid performanceCategory", http.StatusBadRequest)
		return
	}

	data, err := h.s.GetMenuItems(r.Context(), from, to, sortBy, performanceCategory)
	if err != nil {
		http.Error(w, "failed to load menu items", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func parseDateRange(r *http.Request) (time.Time, time.Time, error) {
	const layout = "2006-01-02"
	fromRaw := strings.TrimSpace(r.URL.Query().Get("from"))
	toRaw := strings.TrimSpace(r.URL.Query().Get("to"))

	if fromRaw == "" && toRaw == "" {
		endExclusive := time.Now().UTC().Truncate(24 * time.Hour).Add(24 * time.Hour)
		start := endExclusive.AddDate(0, 0, -30)
		return start, endExclusive, nil
	}
	if fromRaw == "" || toRaw == "" {
		return time.Time{}, time.Time{}, errors.New("both from and to are required when one is provided")
	}

	from, err := time.Parse(layout, fromRaw)
	if err != nil {
		return time.Time{}, time.Time{}, errors.New("invalid from date, expected YYYY-MM-DD")
	}
	to, err := time.Parse(layout, toRaw)
	if err != nil {
		return time.Time{}, time.Time{}, errors.New("invalid to date, expected YYYY-MM-DD")
	}
	endExclusive := to.Add(24 * time.Hour)
	if !endExclusive.After(from) {
		return time.Time{}, time.Time{}, errors.New("to must be greater than or equal to from")
	}

	return from.UTC(), endExclusive.UTC(), nil
}

func (h *Handler) ServeGoogleReviewHTTP(w http.ResponseWriter, r *http.Request) {
	data, err := h.s.GetGoogleReviews()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(data)
}
