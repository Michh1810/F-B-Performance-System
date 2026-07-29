package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"fbperformance/internal/agents/menuidea"
	"fbperformance/internal/store"
)

// fakeIdeaStore satisfies IdeaStore without a real database. UpdateStatus
// and PromoteToMenuItem are tracked independently, since the handler now
// routes "promoted" to PromoteToMenuItem instead of UpdateStatus.
type fakeIdeaStore struct {
	ideas       []menuidea.StoredIdea
	listErr     error
	updateErr   error
	updatedID   uuid.UUID
	updatedTo   string
	updateCalls int

	promoteErr    error
	promoteResult menuidea.StoredIdea
	promotedID    uuid.UUID
	promotedPrice int64
	promotedCogs  int64
	promoteCalls  int

	saveErr error
	saved   []menuidea.StoredIdea
	saveMu  sync.Mutex
}

func (f *fakeIdeaStore) Save(ctx context.Context, idea menuidea.StoredIdea) error {
	if f.saveErr != nil {
		return f.saveErr
	}
	f.saveMu.Lock()
	f.saved = append(f.saved, idea)
	f.saveMu.Unlock()
	return nil
}

func (f *fakeIdeaStore) List(ctx context.Context, status string) ([]menuidea.StoredIdea, error) {
	if f.listErr != nil {
		return nil, f.listErr
	}
	if status == "" {
		return f.ideas, nil
	}
	var filtered []menuidea.StoredIdea
	for _, idea := range f.ideas {
		if idea.Status == status {
			filtered = append(filtered, idea)
		}
	}
	return filtered, nil
}

func (f *fakeIdeaStore) UpdateStatus(ctx context.Context, id uuid.UUID, status string) error {
	f.updateCalls++
	f.updatedID = id
	f.updatedTo = status
	return f.updateErr
}

func (f *fakeIdeaStore) PromoteToMenuItem(ctx context.Context, id uuid.UUID, priceCents, cogsCents int64) (menuidea.StoredIdea, error) {
	f.promoteCalls++
	f.promotedID = id
	f.promotedPrice = priceCents
	f.promotedCogs = cogsCents
	if f.promoteErr != nil {
		return menuidea.StoredIdea{}, f.promoteErr
	}
	return f.promoteResult, nil
}

func newIdeasRouter(h *IdeasHandler) http.Handler {
	r := chi.NewRouter()
	r.Get("/ideas", h.List)
	r.Post("/ideas/run", h.Run)
	r.Patch("/ideas/{id}", h.UpdateStatus)
	return r
}

// fakeIdeaGenerator satisfies menuidea.IdeaGenerator without calling a real
// LLM. Returns candidates keyed by item name, so tests can drive
// per-item behavior (some items produce ideas, some don't, one errors).
type fakeIdeaGenerator struct {
	mu         sync.Mutex
	candidates map[string][]menuidea.IdeaCandidate
	errs       map[string]error
	calledFor  []string
}

func (f *fakeIdeaGenerator) GenerateIdeas(ctx context.Context, item menuidea.MenuItem) ([]menuidea.IdeaCandidate, menuidea.CallLog, error) {
	f.mu.Lock()
	f.calledFor = append(f.calledFor, item.Name)
	f.mu.Unlock()
	return f.candidates[item.Name], menuidea.CallLog{MenuItemID: item.ID, MenuItemName: item.Name}, f.errs[item.Name]
}

// fakeCallLogStore satisfies menuidea.CallLogStore without a real database.
type fakeCallLogStore struct {
	mu   sync.Mutex
	logs []menuidea.CallLog
}

func (f *fakeCallLogStore) Save(ctx context.Context, batchRunID uuid.UUID, log menuidea.CallLog) error {
	f.mu.Lock()
	f.logs = append(f.logs, log)
	f.mu.Unlock()
	return nil
}

func TestIdeasList_NoFilter(t *testing.T) {
	store := &fakeIdeaStore{ideas: []menuidea.StoredIdea{
		{ID: uuid.New(), Status: "new"},
		{ID: uuid.New(), Status: "promoted"},
	}}
	h := NewIdeasHandler(store, nil, nil, nil)
	req := httptest.NewRequest(http.MethodGet, "/ideas", nil)
	rec := httptest.NewRecorder()
	newIdeasRouter(h).ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
}

func TestIdeasList_StatusFilter(t *testing.T) {
	store := &fakeIdeaStore{ideas: []menuidea.StoredIdea{
		{ID: uuid.New(), Status: "new"},
		{ID: uuid.New(), Status: "promoted"},
	}}
	h := NewIdeasHandler(store, nil, nil, nil)
	req := httptest.NewRequest(http.MethodGet, "/ideas?status=promoted", nil)
	rec := httptest.NewRecorder()
	newIdeasRouter(h).ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	if !bytes.Contains(rec.Body.Bytes(), []byte(`"status":"promoted"`)) {
		t.Fatalf("expected only promoted idea in body, got %s", rec.Body.String())
	}
}

func TestIdeasList_InvalidStatusFilter(t *testing.T) {
	h := NewIdeasHandler(&fakeIdeaStore{}, nil, nil, nil)
	req := httptest.NewRequest(http.MethodGet, "/ideas?status=bogus", nil)
	rec := httptest.NewRecorder()
	newIdeasRouter(h).ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
}

func TestIdeasUpdateStatus_NonPromotedStatuses(t *testing.T) {
	for _, status := range []string{"new", "reviewed", "dismissed"} {
		t.Run(status, func(t *testing.T) {
			fake := &fakeIdeaStore{}
			h := NewIdeasHandler(fake, nil, nil, nil)
			id := uuid.New()
			req := httptest.NewRequest(http.MethodPatch, "/ideas/"+id.String(), bytes.NewBufferString(`{"status":"`+status+`"}`))
			rec := httptest.NewRecorder()
			newIdeasRouter(h).ServeHTTP(rec, req)

			if rec.Code != http.StatusOK {
				t.Fatalf("status = %d, want %d, body=%s", rec.Code, http.StatusOK, rec.Body.String())
			}
			if fake.updatedID != id || fake.updatedTo != status {
				t.Fatalf("UpdateStatus called with (%v, %q), want (%v, %q)", fake.updatedID, fake.updatedTo, id, status)
			}
			if fake.promoteCalls != 0 {
				t.Fatalf("expected PromoteToMenuItem not called for status %q", status)
			}
		})
	}
}

func TestIdeasUpdateStatus_Promoted_RoutesToPromoteToMenuItem(t *testing.T) {
	fake := &fakeIdeaStore{promoteResult: menuidea.StoredIdea{Status: "promoted"}}
	h := NewIdeasHandler(fake, nil, nil, nil)
	id := uuid.New()
	req := httptest.NewRequest(http.MethodPatch, "/ideas/"+id.String(), bytes.NewBufferString(`{"status":"promoted","price_cents":1200,"cogs_cents":400}`))
	rec := httptest.NewRecorder()
	newIdeasRouter(h).ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body=%s", rec.Code, http.StatusOK, rec.Body.String())
	}
	if fake.promoteCalls != 1 {
		t.Fatalf("expected exactly one PromoteToMenuItem call, got %d", fake.promoteCalls)
	}
	if fake.promotedID != id {
		t.Fatalf("PromoteToMenuItem called with id %v, want %v", fake.promotedID, id)
	}
	if fake.promotedPrice != 1200 || fake.promotedCogs != 400 {
		t.Fatalf("PromoteToMenuItem called with (price=%d, cogs=%d), want (1200, 400)", fake.promotedPrice, fake.promotedCogs)
	}
	if fake.updateCalls != 0 {
		t.Fatalf("expected UpdateStatus not called when status is promoted, got %d calls", fake.updateCalls)
	}
}

func TestIdeasUpdateStatus_Promoted_MissingPriceIsBadRequest(t *testing.T) {
	fake := &fakeIdeaStore{promoteErr: store.ErrPriceRequiredToPromote}
	h := NewIdeasHandler(fake, nil, nil, nil)
	req := httptest.NewRequest(http.MethodPatch, "/ideas/"+uuid.New().String(), bytes.NewBufferString(`{"status":"promoted"}`))
	rec := httptest.NewRecorder()
	newIdeasRouter(h).ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d, body=%s", rec.Code, http.StatusBadRequest, rec.Body.String())
	}
}

func TestIdeasUpdateStatus_Promoted_NotFound(t *testing.T) {
	fake := &fakeIdeaStore{promoteErr: store.ErrIdeaNotFound}
	h := NewIdeasHandler(fake, nil, nil, nil)
	req := httptest.NewRequest(http.MethodPatch, "/ideas/"+uuid.New().String(), bytes.NewBufferString(`{"status":"promoted","price_cents":1000,"cogs_cents":300}`))
	rec := httptest.NewRecorder()
	newIdeasRouter(h).ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusNotFound)
	}
}

func TestIdeasUpdateStatus_Promoted_ResponseIncludesCreatedMenuItemID(t *testing.T) {
	itemID := uuid.New()
	fake := &fakeIdeaStore{promoteResult: menuidea.StoredIdea{Status: "promoted", CreatedMenuItemID: &itemID}}
	h := NewIdeasHandler(fake, nil, nil, nil)
	req := httptest.NewRequest(http.MethodPatch, "/ideas/"+uuid.New().String(), bytes.NewBufferString(`{"status":"promoted","price_cents":1200,"cogs_cents":400}`))
	rec := httptest.NewRecorder()
	newIdeasRouter(h).ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	if !bytes.Contains(rec.Body.Bytes(), []byte(itemID.String())) {
		t.Fatalf("expected response to include created_menu_item_id %s, got %s", itemID, rec.Body.String())
	}
}

func TestIdeasUpdateStatus_InvalidStatus(t *testing.T) {
	h := NewIdeasHandler(&fakeIdeaStore{}, nil, nil, nil)
	req := httptest.NewRequest(http.MethodPatch, "/ideas/"+uuid.New().String(), bytes.NewBufferString(`{"status":"bogus"}`))
	rec := httptest.NewRecorder()
	newIdeasRouter(h).ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
}

func TestIdeasUpdateStatus_InvalidID(t *testing.T) {
	h := NewIdeasHandler(&fakeIdeaStore{}, nil, nil, nil)
	req := httptest.NewRequest(http.MethodPatch, "/ideas/not-a-uuid", bytes.NewBufferString(`{"status":"reviewed"}`))
	rec := httptest.NewRecorder()
	newIdeasRouter(h).ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
}

func TestIdeasUpdateStatus_NotFound(t *testing.T) {
	fake := &fakeIdeaStore{updateErr: store.ErrIdeaNotFound}
	h := NewIdeasHandler(fake, nil, nil, nil)
	req := httptest.NewRequest(http.MethodPatch, "/ideas/"+uuid.New().String(), bytes.NewBufferString(`{"status":"reviewed"}`))
	rec := httptest.NewRecorder()
	newIdeasRouter(h).ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusNotFound)
	}
}

func TestIdeasRun_ScansActiveItemsAndPersistsIdeas(t *testing.T) {
	itemA := uuid.New()
	itemB := uuid.New()
	lister := &fakeMenuItemLister{items: []menuidea.MenuItem{
		{ID: itemA, Name: "Bulgogi Bowl", Category: "Entree"},
		{ID: itemB, Name: "Sikhye", Category: "Beverage"},
	}}
	generator := &fakeIdeaGenerator{candidates: map[string][]menuidea.IdeaCandidate{
		"Bulgogi Bowl": {{Kind: "tweak", Name: "Bulgogi Fries"}},
		// Sikhye intentionally produces no ideas — exercises the
		// zero-candidates path within a single Run call.
	}}
	callLogs := &fakeCallLogStore{}
	ideas := &fakeIdeaStore{}

	h := NewIdeasHandler(ideas, lister, generator, callLogs)
	req := httptest.NewRequest(http.MethodPost, "/ideas/run", nil)
	rec := httptest.NewRecorder()
	newIdeasRouter(h).ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body=%s", rec.Code, http.StatusOK, rec.Body.String())
	}

	var resp runSummary
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode response: %v, body=%s", err, rec.Body.String())
	}
	if resp.MenuItemsScanned != 2 {
		t.Fatalf("menu_items_scanned = %d, want 2", resp.MenuItemsScanned)
	}
	if resp.IdeasGenerated != 1 {
		t.Fatalf("ideas_generated = %d, want 1", resp.IdeasGenerated)
	}
	if resp.BatchRunID == "" {
		t.Fatalf("expected non-empty batch_run_id")
	}

	if len(ideas.saved) != 1 || ideas.saved[0].Name != "Bulgogi Fries" {
		t.Fatalf("expected one saved idea %q, got %+v", "Bulgogi Fries", ideas.saved)
	}
	if len(callLogs.logs) != 2 {
		t.Fatalf("expected a call log persisted per scanned item, got %d", len(callLogs.logs))
	}
}

func TestIdeasRun_MenuItemListErrorIsInternalServerError(t *testing.T) {
	lister := &fakeMenuItemLister{err: context.DeadlineExceeded}
	h := NewIdeasHandler(&fakeIdeaStore{}, lister, &fakeIdeaGenerator{}, &fakeCallLogStore{})
	req := httptest.NewRequest(http.MethodPost, "/ideas/run", nil)
	rec := httptest.NewRecorder()
	newIdeasRouter(h).ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusInternalServerError)
	}
}
