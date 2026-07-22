package handlers

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"

	"fbperformance/internal/agents/financial"
	"fbperformance/internal/agents/manager"
	"fbperformance/internal/agents/orchestrator"
	"fbperformance/internal/agents/trend"
	"fbperformance/internal/services/llm"
)

// noopSnapshotStore satisfies trend.SnapshotStore without ever being
// invoked — these tests only exercise the handler's validation path, which
// rejects requests before the orchestrator (and therefore the trend agent)
// is ever called.
type noopSnapshotStore struct{}

func (noopSnapshotStore) Recent(ctx context.Context, menuItemID uuid.UUID, limit int) ([]trend.Snapshot, error) {
	return nil, nil
}

func (noopSnapshotStore) Save(ctx context.Context, snap trend.Snapshot) error {
	return nil
}

// noopEmbedder/noopSignalSearcher satisfy trend.Embedder/trend.SignalSearcher
// for the same reason — never invoked by these validation-path tests.
type noopEmbedder struct{}

func (noopEmbedder) Embed(ctx context.Context, model, text string, opts llm.EmbedOptions) ([]float32, error) {
	return nil, nil
}

type noopSignalSearcher struct{}

func (noopSignalSearcher) SearchSimilar(ctx context.Context, embedding []float32, limit int, since time.Time) ([]trend.Signal, error) {
	return nil, nil
}

func newTestHandler() *RecommendationHandler {
	llmClient := llm.NewClient("")
	trendAgent := trend.NewAgent(llmClient, noopEmbedder{}, noopSignalSearcher{}, noopSnapshotStore{}, "test-model", "embed-model", 30)
	financialAgent := financial.NewAgent(llmClient, "test-model")
	managerAgent := manager.NewAgent(llmClient, "test-model")
	o := orchestrator.New(trendAgent, financialAgent, managerAgent)
	return NewRecommendationHandler(o)
}

func postJSON(h http.Handler, body string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodPost, "/api/ai/recommendation", bytes.NewBufferString(body))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

func TestServeHTTP_MissingMenuItemID(t *testing.T) {
	rec := postJSON(newTestHandler(), `{"item_name": "Spicy Chicken Sandwich"}`)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
}

func TestServeHTTP_InvalidMenuItemID(t *testing.T) {
	rec := postJSON(newTestHandler(), `{"item_name": "Spicy Chicken Sandwich", "menu_item_id": "not-a-uuid"}`)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
}

func TestServeHTTP_MissingItemName(t *testing.T) {
	rec := postJSON(newTestHandler(), `{"menu_item_id": "`+uuid.New().String()+`"}`)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
}
