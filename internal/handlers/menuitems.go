package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/google/uuid"
)

// activeMenuItemResponse is the wire shape for one active menu item —
// declared locally (rather than reusing menuidea.MenuItem directly) because
// that struct carries no json tags, having only ever needed Go-side
// consumption before this handler existed.
type activeMenuItemResponse struct {
	ID       uuid.UUID `json:"id"`
	Name     string    `json:"name"`
	Category string    `json:"category"`
}

// MenuItemsHandler exposes the restaurant's active menu — the same
// ListActive data set the Menu Idea Agent scans — for the frontend to
// summarize and browse, via the MenuItemLister interface already declared
// in trendhashtags.go.
type MenuItemsHandler struct {
	menuItems MenuItemLister
}

func NewMenuItemsHandler(menuItems MenuItemLister) *MenuItemsHandler {
	return &MenuItemsHandler{menuItems: menuItems}
}

// ListActive handles GET /api/menu-items/active.
func (h *MenuItemsHandler) ListActive(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	items, err := h.menuItems.ListActive(r.Context())
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	resp := make([]activeMenuItemResponse, len(items))
	for i, item := range items {
		resp[i] = activeMenuItemResponse{ID: item.ID, Name: item.Name, Category: item.Category}
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(resp)
}
