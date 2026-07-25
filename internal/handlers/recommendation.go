package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/google/uuid"

	"fbperformance/internal/agents/orchestrator"
)

type RecommendationHandler struct {
	orchestrator *orchestrator.Orchestrator
}

func NewRecommendationHandler(o *orchestrator.Orchestrator) *RecommendationHandler {
	return &RecommendationHandler{orchestrator: o}
}

func (h *RecommendationHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		json.NewEncoder(w).Encode(map[string]string{"error": "method not allowed"})
		return
	}

	var req orchestrator.Request
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.ItemName == "" || req.MenuItemID == uuid.Nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "item_name and a valid menu_item_id are required"})
		return
	}

	resp, err := h.orchestrator.GetRecommendation(r.Context(), req)
	if err != nil {
		w.WriteHeader(http.StatusBadGateway)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(resp)
}
