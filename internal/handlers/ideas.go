package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"fbperformance/internal/agents/menuidea"
	"fbperformance/internal/store"
)

// validIdeaStatuses are the allowed values for IdeasHandler.UpdateStatus.
// "promoted" on a "tweak"-kind idea creates a real menu_items row (see
// IdeaStore.PromoteToMenuItem) — it is the one status with a real
// downstream effect; the others are pure review-workflow labels.
var validIdeaStatuses = map[string]bool{
	"new":       true,
	"reviewed":  true,
	"dismissed": true,
	"promoted":  true,
}

// IdeaStore is the subset of menu-idea storage this handler depends on —
// declared locally (rather than depending on *store.MenuIdeaStore directly)
// so tests can exercise the handler with a hand-written fake instead of a
// real Postgres connection, matching this repo's convention (see
// RecommendationHandler / recommendation_test.go). *store.MenuIdeaStore
// satisfies this as-is.
type IdeaStore interface {
	List(ctx context.Context, status string) ([]menuidea.StoredIdea, error)
	UpdateStatus(ctx context.Context, id uuid.UUID, status string) error
	PromoteToMenuItem(ctx context.Context, id uuid.UUID, priceCents, cogsCents int64) (menuidea.StoredIdea, error)
}

// IdeasHandler exposes Menu Idea Agent output for human review: List
// returns generated ideas (optionally filtered by status), UpdateStatus
// records a human's review verdict on one idea.
type IdeasHandler struct {
	ideas IdeaStore
}

func NewIdeasHandler(ideas IdeaStore) *IdeasHandler {
	return &IdeasHandler{ideas: ideas}
}

// List handles GET /api/ai/ideas, optionally filtered by ?status=.
func (h *IdeasHandler) List(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	status := r.URL.Query().Get("status")
	if status != "" && !validIdeaStatuses[status] {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "invalid status filter"})
		return
	}

	ideas, err := h.ideas.List(r.Context(), status)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(ideas)
}

// updateStatusRequest is the request body for UpdateStatus. PriceCents and
// CogsCents are only used (and only required) when Status is "promoted"
// and the idea is "tweak"-kind — see IdeaStore.PromoteToMenuItem.
type updateStatusRequest struct {
	Status     string `json:"status"`
	PriceCents int64  `json:"price_cents"`
	CogsCents  int64  `json:"cogs_cents"`
}

// UpdateStatus handles PATCH /api/ai/ideas/{id}, transitioning an idea's
// review status. "promoted" is routed to PromoteToMenuItem instead of
// UpdateStatus, since it can have a real side effect (creating a menu_items
// row) that the other statuses deliberately don't have.
func (h *IdeasHandler) UpdateStatus(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "invalid idea id"})
		return
	}

	var req updateStatusRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || !validIdeaStatuses[req.Status] {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "status must be one of new, reviewed, dismissed, promoted"})
		return
	}

	if req.Status == "promoted" {
		idea, err := h.ideas.PromoteToMenuItem(r.Context(), id, req.PriceCents, req.CogsCents)
		if err != nil {
			switch {
			case errors.Is(err, store.ErrIdeaNotFound):
				w.WriteHeader(http.StatusNotFound)
				json.NewEncoder(w).Encode(map[string]string{"error": "idea not found"})
			case errors.Is(err, store.ErrPriceRequiredToPromote):
				w.WriteHeader(http.StatusBadRequest)
				json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
			default:
				w.WriteHeader(http.StatusInternalServerError)
				json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
			}
			return
		}
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(idea)
		return
	}

	if err := h.ideas.UpdateStatus(r.Context(), id, req.Status); err != nil {
		if errors.Is(err, store.ErrIdeaNotFound) {
			w.WriteHeader(http.StatusNotFound)
			json.NewEncoder(w).Encode(map[string]string{"error": "idea not found"})
			return
		}
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": req.Status})
}
