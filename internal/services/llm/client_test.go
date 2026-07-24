package llm

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestEmbed(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1beta/models/test-embed-model:embedContent" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}

		// Decode into a generic map, not embedContentRequest, so this test
		// actually catches a regression to nesting taskType/
		// outputDimensionality under an "embedContentConfig" object — the
		// real bug found live against the Gemini API, where that nesting
		// silently ignores outputDimensionality and returns the model's
		// native 3072-dim output instead of the requested size.
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatalf("read request body: %v", err)
		}
		var raw map[string]any
		if err := json.Unmarshal(body, &raw); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		if _, ok := raw["embedContentConfig"]; ok {
			t.Fatalf("taskType/outputDimensionality must be top-level fields, not nested under embedContentConfig: %s", body)
		}
		if raw["taskType"] != "RETRIEVAL_QUERY" {
			t.Fatalf("top-level taskType = %v, want RETRIEVAL_QUERY", raw["taskType"])
		}
		if raw["outputDimensionality"] != float64(3) {
			t.Fatalf("top-level outputDimensionality = %v, want 3", raw["outputDimensionality"])
		}

		var req embedContentRequest
		if err := json.Unmarshal(body, &req); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		if len(req.Content.Parts) != 1 || req.Content.Parts[0].Text != "fusion coffee" {
			t.Fatalf("unexpected request content: %+v", req.Content)
		}

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"embedding": {"values": [0.1, 0.2, 0.3]}}`))
	}))
	defer server.Close()

	client := NewClient("test-key", WithBaseURL(server.URL))
	values, err := client.Embed(context.Background(), "test-embed-model", "fusion coffee", EmbedOptions{
		TaskType:             "RETRIEVAL_QUERY",
		OutputDimensionality: 3,
	})
	if err != nil {
		t.Fatalf("Embed() error = %v, want nil", err)
	}
	if len(values) != 3 || values[0] != 0.1 {
		t.Fatalf("values = %v, want [0.1 0.2 0.3]", values)
	}
}

func TestEmbed_ErrorStatus(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
		_, _ = w.Write([]byte(`{"error": "overloaded"}`))
	}))
	defer server.Close()

	client := NewClient("test-key", WithBaseURL(server.URL))
	if _, err := client.Embed(context.Background(), "test-embed-model", "text", EmbedOptions{}); err == nil {
		t.Fatalf("expected error on non-200 status")
	}
}

func TestEmbed_NoEmbeddingReturned(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"embedding": {"values": []}}`))
	}))
	defer server.Close()

	client := NewClient("test-key", WithBaseURL(server.URL))
	if _, err := client.Embed(context.Background(), "test-embed-model", "text", EmbedOptions{}); err == nil {
		t.Fatalf("expected error when no embedding values are returned")
	}
}
