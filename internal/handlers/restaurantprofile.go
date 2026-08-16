package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"

	"fbperformance/internal/store"
)

// RestaurantProfileStore is the subset of restaurant-profile storage this
// handler depends on — declared locally so tests can use a hand-written
// fake instead of a real Postgres connection. *store.RestaurantProfileStore
// satisfies this as-is.
type RestaurantProfileStore interface {
	Get(ctx context.Context) (store.RestaurantProfile, error)
	Upsert(ctx context.Context, description string) (store.RestaurantProfile, error)
}

// RestaurantProfileHandler exposes this restaurant's free-text profile: Get
// reads it, Upsert sets it. TrendHashtagsHandler.Generate reads whatever is
// stored here to ground its hashtag suggestions.
type RestaurantProfileHandler struct {
	profiles RestaurantProfileStore
}

func NewRestaurantProfileHandler(profiles RestaurantProfileStore) *RestaurantProfileHandler {
	return &RestaurantProfileHandler{profiles: profiles}
}

// Get handles GET /api/restaurant-profile.
func (h *RestaurantProfileHandler) Get(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	profile, err := h.profiles.Get(r.Context())
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(profile)
}

type upsertProfileRequest struct {
	Description string `json:"description"`
}

// Upsert handles PUT /api/restaurant-profile.
func (h *RestaurantProfileHandler) Upsert(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	var req upsertProfileRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || strings.TrimSpace(req.Description) == "" {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "description is required"})
		return
	}

	profile, err := h.profiles.Upsert(r.Context(), req.Description)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(profile)
}
