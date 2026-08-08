package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"fbperformance/internal/agents/hashtagsuggest"
	"fbperformance/internal/agents/menuidea"
	"fbperformance/internal/store"
)

// validHashtagSuggestionFilters are the allowed ?status= filter values for
// List — a superset of the transitionable statuses, since "pending" is a
// valid thing to filter by even though it's not something UpdateStatus can
// set a suggestion back to.
var validHashtagSuggestionFilters = map[string]bool{
	"pending":  true,
	"approved": true,
	"rejected": true,
}

// validHashtagSuggestionTransitions are the statuses UpdateStatus may set —
// a suggestion starts "pending" on generation and can only move forward to
// approved or rejected, never back.
var validHashtagSuggestionTransitions = map[string]bool{
	"approved": true,
	"rejected": true,
}

// MenuItemLister is the subset of menu-item storage this handler depends
// on, to ground hashtag suggestions in the current active menu.
type MenuItemLister interface {
	ListActive(ctx context.Context) ([]menuidea.MenuItem, error)
}

// HashtagSuggester is the subset of hashtagsuggest this handler depends on.
type HashtagSuggester interface {
	Suggest(ctx context.Context, in hashtagsuggest.Input) ([]string, error)
}

// HashtagSuggestionStore is the subset of suggestion storage this handler
// depends on.
type HashtagSuggestionStore interface {
	Save(ctx context.Context, sourceDescription string, hashtags []string) (store.HashtagSuggestion, error)
	List(ctx context.Context, status string) ([]store.HashtagSuggestion, error)
	UpdateStatus(ctx context.Context, id uuid.UUID, status string) error
	// LatestApproved returns the hashtags cmd/trend-ingest would sweep right
	// now — used after a successful approve to trigger a real sweep of
	// exactly that list, rather than re-deriving it locally.
	LatestApproved(ctx context.Context) ([]string, error)
}

// TrendIngestTrigger is the subset of trendingest.Service this handler
// depends on to kick off a real TikTok sweep the moment a suggestion is
// approved, instead of waiting for the next scheduled cmd/trend-ingest run.
type TrendIngestTrigger interface {
	// TriggerAsync starts a sweep in the background and returns
	// immediately. Returns false (a no-op) if one is already running.
	TriggerAsync(hashtags []string) bool
}

// TrendHashtagsHandler drives the generate-then-review workflow for
// cmd/trend-ingest's hashtag list: Generate asks the LLM for suggestions
// grounded in the current restaurant profile + menu, List surfaces them for
// review, and UpdateStatus records a human's approve/reject verdict —
// approving also triggers a real background sweep via trendIngest.
type TrendHashtagsHandler struct {
	profiles    RestaurantProfileStore
	menuItems   MenuItemLister
	suggester   HashtagSuggester
	suggestions HashtagSuggestionStore
	trendIngest TrendIngestTrigger
}

func NewTrendHashtagsHandler(profiles RestaurantProfileStore, menuItems MenuItemLister, suggester HashtagSuggester, suggestions HashtagSuggestionStore, trendIngest TrendIngestTrigger) *TrendHashtagsHandler {
	return &TrendHashtagsHandler{
		profiles:    profiles,
		menuItems:   menuItems,
		suggester:   suggester,
		suggestions: suggestions,
		trendIngest: trendIngest,
	}
}

// Generate handles POST /api/trend-hashtags/suggestions: it loads the
// current restaurant profile and active menu, asks the LLM for a hashtag
// list, and persists it as a new "pending" suggestion.
func (h *TrendHashtagsHandler) Generate(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	profile, err := h.profiles.Get(r.Context())
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}
	if profile.Description == "" {
		w.WriteHeader(http.StatusUnprocessableEntity)
		json.NewEncoder(w).Encode(map[string]string{"error": "set a restaurant profile description (PUT /api/restaurant-profile) before generating hashtag suggestions"})
		return
	}

	items, err := h.menuItems.ListActive(r.Context())
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}
	menuItemRefs := make([]hashtagsuggest.MenuItemRef, len(items))
	for i, item := range items {
		menuItemRefs[i] = hashtagsuggest.MenuItemRef{Name: item.Name, Category: item.Category}
	}

	hashtags, err := h.suggester.Suggest(r.Context(), hashtagsuggest.Input{
		Description: profile.Description,
		MenuItems:   menuItemRefs,
	})
	if err != nil {
		w.WriteHeader(http.StatusBadGateway)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	suggestion, err := h.suggestions.Save(r.Context(), profile.Description, hashtags)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(suggestion)
}

// List handles GET /api/trend-hashtags/suggestions, optionally filtered by
// ?status=.
func (h *TrendHashtagsHandler) List(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	status := r.URL.Query().Get("status")
	if status != "" && !validHashtagSuggestionFilters[status] {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "invalid status filter"})
		return
	}

	suggestions, err := h.suggestions.List(r.Context(), status)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(suggestions)
}

type saveManualHashtagsRequest struct {
	Hashtags []string `json:"hashtags"`
}

// SaveManual handles POST /api/trend-hashtags/suggestions/manual: persists a
// human-edited hashtag list (added/removed on the dashboard, not an LLM
// Generate call) as a new "pending" suggestion, so it flows through the same
// approve/reject review UI — and the same LatestApproved() gate — as a
// generated one before cmd/trend-ingest can sweep it.
func (h *TrendHashtagsHandler) SaveManual(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	var req saveManualHashtagsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || len(req.Hashtags) == 0 {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "hashtags must be a non-empty array"})
		return
	}

	suggestion, err := h.suggestions.Save(r.Context(), "manual edit", req.Hashtags)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(suggestion)
}

type updateHashtagSuggestionStatusRequest struct {
	Status string `json:"status"`
}

// UpdateStatus handles PATCH /api/trend-hashtags/suggestions/{id},
// approving or rejecting a suggestion.
func (h *TrendHashtagsHandler) UpdateStatus(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "invalid suggestion id"})
		return
	}

	var req updateHashtagSuggestionStatusRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || !validHashtagSuggestionTransitions[req.Status] {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "status must be approved or rejected"})
		return
	}

	if err := h.suggestions.UpdateStatus(r.Context(), id, req.Status); err != nil {
		if errors.Is(err, store.ErrHashtagSuggestionNotFound) {
			w.WriteHeader(http.StatusNotFound)
			json.NewEncoder(w).Encode(map[string]string{"error": "suggestion not found"})
			return
		}
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	// Approving is the signal that this hashtag list is ready to sweep for
	// real — trigger cmd/trend-ingest's logic in the background instead of
	// waiting for the next scheduled run. Re-reading LatestApproved (rather
	// than trusting the request body) guarantees we sweep exactly what the
	// approve/reject gate now considers current, even under a race with
	// another concurrent approval. ingestTriggered is false both when
	// nothing needed sweeping and when a sweep was already running — the
	// frontend polls GET /api/trend-ingest/status either way.
	var ingestTriggered bool
	if req.Status == "approved" && h.trendIngest != nil {
		if hashtags, err := h.suggestions.LatestApproved(r.Context()); err == nil {
			ingestTriggered = h.trendIngest.TriggerAsync(hashtags)
		}
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]any{"status": req.Status, "ingest_triggered": ingestTriggered})
}
