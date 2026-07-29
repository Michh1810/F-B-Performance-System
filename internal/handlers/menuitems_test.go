package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"

	"fbperformance/internal/agents/menuidea"
)

type fakeMenuItemLister struct {
	items []menuidea.MenuItem
	err   error
}

func (f *fakeMenuItemLister) ListActive(ctx context.Context) ([]menuidea.MenuItem, error) {
	if f.err != nil {
		return nil, f.err
	}
	return f.items, nil
}

func TestMenuItemsListActive_OK(t *testing.T) {
	lister := &fakeMenuItemLister{items: []menuidea.MenuItem{
		{ID: uuid.New(), Name: "Birria Tacos", Category: "Entree"},
		{ID: uuid.New(), Name: "Guacamole", Category: "Appetizer"},
	}}
	h := NewMenuItemsHandler(lister)

	req := httptest.NewRequest(http.MethodGet, "/api/menu-items/active", nil)
	w := httptest.NewRecorder()
	h.ListActive(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
	}

	var got []activeMenuItemResponse
	if err := json.NewDecoder(w.Body).Decode(&got); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("len(got) = %d, want 2", len(got))
	}
	if got[0].Name != "Birria Tacos" || got[0].Category != "Entree" {
		t.Errorf("got[0] = %+v, want name=Birria Tacos category=Entree", got[0])
	}
}

func TestMenuItemsListActive_StoreError(t *testing.T) {
	lister := &fakeMenuItemLister{err: context.DeadlineExceeded}
	h := NewMenuItemsHandler(lister)

	req := httptest.NewRequest(http.MethodGet, "/api/menu-items/active", nil)
	w := httptest.NewRecorder()
	h.ListActive(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusInternalServerError)
	}
}
