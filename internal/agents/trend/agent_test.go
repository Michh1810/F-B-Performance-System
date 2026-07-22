package trend

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"

	"fbperformance/internal/services/llm"
)

// fakeGenerator returns responses[call]/errs[call] for the call'th
// invocation (indices past the end of either slice yield zero values),
// letting tests give different behavior to Analyze's sequential Generate
// calls (classify, then narrative).
type fakeGenerator struct {
	responses []string
	errs      []error
	calls     int
}

func (f *fakeGenerator) Generate(ctx context.Context, model, prompt string, opts llm.GenerateOptions) (string, error) {
	i := f.calls
	f.calls++
	var resp string
	var err error
	if i < len(f.responses) {
		resp = f.responses[i]
	}
	if i < len(f.errs) {
		err = f.errs[i]
	}
	return resp, err
}

type fakeEmbedder struct {
	embedding []float32
	err       error
	calls     int
}

func (f *fakeEmbedder) Embed(ctx context.Context, model, text string, opts llm.EmbedOptions) ([]float32, error) {
	f.calls++
	return f.embedding, f.err
}

type fakeSignalSearcher struct {
	signals []Signal
	err     error
	calls   int
}

func (f *fakeSignalSearcher) SearchSimilar(ctx context.Context, embedding []float32, limit int, since time.Time) ([]Signal, error) {
	f.calls++
	if f.err != nil {
		return nil, f.err
	}
	return f.signals, nil
}

type fakeSnapshotStore struct {
	recent    []Snapshot
	recentErr error
	saveErr   error
	saved     []Snapshot
}

func (f *fakeSnapshotStore) Recent(ctx context.Context, menuItemID uuid.UUID, limit int) ([]Snapshot, error) {
	if f.recentErr != nil {
		return nil, f.recentErr
	}
	if limit < len(f.recent) {
		return f.recent[:limit], nil
	}
	return f.recent, nil
}

func (f *fakeSnapshotStore) Save(ctx context.Context, snap Snapshot) error {
	f.saved = append(f.saved, snap)
	return f.saveErr
}

const validClassification = `{"sentiment_label":"positive","sentiment_score":0.9,"related_food_trends":["birria"]}`

func oneSignal(views int64) []Signal {
	return []Signal{{ExternalID: "1", Caption: "test", Hashtags: []string{"foodtiktok"}, ViewCount: views, LikeCount: views / 10}}
}

func TestAnalyze_NoRelevantSignal(t *testing.T) {
	gen := &fakeGenerator{}
	embedder := &fakeEmbedder{embedding: []float32{0.1, 0.2}}
	searcher := &fakeSignalSearcher{signals: nil}
	snapshots := &fakeSnapshotStore{}
	agent := NewAgent(gen, embedder, searcher, snapshots, "test-model", "embed-model", 30)

	result, err := agent.Analyze(context.Background(), Input{MenuItemID: uuid.New(), ItemName: "Fusion Coffee"})
	if err != nil {
		t.Fatalf("Analyze() error = %v, want nil", err)
	}
	if gen.calls != 0 {
		t.Fatalf("expected Generator not called with zero signals, got %d calls", gen.calls)
	}
	if len(snapshots.saved) != 0 {
		t.Fatalf("expected no snapshot saved with zero signals, got %d", len(snapshots.saved))
	}
	if result.SentimentLabel != "unknown" {
		t.Fatalf("SentimentLabel = %q, want %q", result.SentimentLabel, "unknown")
	}
}

func TestAnalyze_EmbedFailureIsGraceful(t *testing.T) {
	gen := &fakeGenerator{}
	embedder := &fakeEmbedder{err: errors.New("embedding api down")}
	searcher := &fakeSignalSearcher{signals: oneSignal(1000)}
	snapshots := &fakeSnapshotStore{}
	agent := NewAgent(gen, embedder, searcher, snapshots, "test-model", "embed-model", 30)

	result, err := agent.Analyze(context.Background(), Input{MenuItemID: uuid.New(), ItemName: "Fusion Coffee"})
	if err != nil {
		t.Fatalf("Analyze() error = %v, want nil (embed failure should degrade gracefully)", err)
	}
	if searcher.calls != 0 {
		t.Fatalf("expected SearchSimilar not called after embed failure, got %d calls", searcher.calls)
	}
	if result.SentimentLabel != "unknown" {
		t.Fatalf("SentimentLabel = %q, want %q", result.SentimentLabel, "unknown")
	}
}

func TestAnalyze_SearchFailureIsGraceful(t *testing.T) {
	gen := &fakeGenerator{}
	embedder := &fakeEmbedder{embedding: []float32{0.1}}
	searcher := &fakeSignalSearcher{err: errors.New("db down")}
	snapshots := &fakeSnapshotStore{}
	agent := NewAgent(gen, embedder, searcher, snapshots, "test-model", "embed-model", 30)

	result, err := agent.Analyze(context.Background(), Input{MenuItemID: uuid.New(), ItemName: "Fusion Coffee"})
	if err != nil {
		t.Fatalf("Analyze() error = %v, want nil (search failure should degrade gracefully)", err)
	}
	if result.SentimentLabel != "unknown" {
		t.Fatalf("SentimentLabel = %q, want %q", result.SentimentLabel, "unknown")
	}
}

func TestAnalyze_ClassifyFailureDegradesGracefully(t *testing.T) {
	gen := &fakeGenerator{
		responses: []string{"", "narrative text"},
		errs:      []error{errors.New("classification hiccup"), nil},
	}
	embedder := &fakeEmbedder{embedding: []float32{0.1}}
	searcher := &fakeSignalSearcher{signals: oneSignal(1000)}
	snapshots := &fakeSnapshotStore{}
	agent := NewAgent(gen, embedder, searcher, snapshots, "test-model", "embed-model", 30)

	result, err := agent.Analyze(context.Background(), Input{MenuItemID: uuid.New(), ItemName: "Fusion Coffee"})
	if err != nil {
		t.Fatalf("Analyze() error = %v, want nil (classify failure should degrade, not abort)", err)
	}
	if gen.calls != 2 {
		t.Fatalf("expected both classify and narrative calls to happen, got %d calls", gen.calls)
	}
	if result.SentimentLabel != "unknown" {
		t.Fatalf("SentimentLabel = %q, want %q after classify failure", result.SentimentLabel, "unknown")
	}
	if result.Summary != "narrative text" {
		t.Fatalf("Summary = %q, want narrative to still succeed", result.Summary)
	}
	if len(snapshots.saved) != 1 {
		t.Fatalf("expected snapshot still saved after classify failure, got %d", len(snapshots.saved))
	}
}

func TestAnalyze_NarrativeFailureHardFails(t *testing.T) {
	gen := &fakeGenerator{
		responses: []string{validClassification, ""},
		errs:      []error{nil, errors.New("narrative down")},
	}
	embedder := &fakeEmbedder{embedding: []float32{0.1}}
	searcher := &fakeSignalSearcher{signals: oneSignal(1000)}
	snapshots := &fakeSnapshotStore{}
	agent := NewAgent(gen, embedder, searcher, snapshots, "test-model", "embed-model", 30)

	_, err := agent.Analyze(context.Background(), Input{MenuItemID: uuid.New(), ItemName: "Fusion Coffee"})
	if err == nil {
		t.Fatalf("expected Analyze() to hard-fail when narrative synthesis errors")
	}
	if len(snapshots.saved) != 0 {
		t.Fatalf("expected no snapshot saved when narrative fails, got %d", len(snapshots.saved))
	}
}

func TestAnalyze_RecentReadErrorHardFails(t *testing.T) {
	gen := &fakeGenerator{responses: []string{validClassification}}
	embedder := &fakeEmbedder{embedding: []float32{0.1}}
	searcher := &fakeSignalSearcher{signals: oneSignal(1000)}
	snapshots := &fakeSnapshotStore{recentErr: errors.New("db down")}
	agent := NewAgent(gen, embedder, searcher, snapshots, "test-model", "embed-model", 30)

	_, err := agent.Analyze(context.Background(), Input{MenuItemID: uuid.New(), ItemName: "Fusion Coffee"})
	if err == nil {
		t.Fatalf("expected Analyze() to hard-fail when reading the previous snapshot errors")
	}
}

func TestAnalyze_SaveErrorHardFails(t *testing.T) {
	gen := &fakeGenerator{responses: []string{validClassification, "narrative"}}
	embedder := &fakeEmbedder{embedding: []float32{0.1}}
	searcher := &fakeSignalSearcher{signals: oneSignal(1000)}
	snapshots := &fakeSnapshotStore{saveErr: errors.New("db down")}
	agent := NewAgent(gen, embedder, searcher, snapshots, "test-model", "embed-model", 30)

	_, err := agent.Analyze(context.Background(), Input{MenuItemID: uuid.New(), ItemName: "Fusion Coffee"})
	if err == nil {
		t.Fatalf("expected Analyze() to hard-fail when saving the new snapshot errors")
	}
}

func TestAnalyze_NoPriorSnapshot(t *testing.T) {
	gen := &fakeGenerator{responses: []string{validClassification, "narrative"}}
	embedder := &fakeEmbedder{embedding: []float32{0.1}}
	searcher := &fakeSignalSearcher{signals: oneSignal(1000)}
	snapshots := &fakeSnapshotStore{}
	agent := NewAgent(gen, embedder, searcher, snapshots, "test-model", "embed-model", 30)

	result, err := agent.Analyze(context.Background(), Input{MenuItemID: uuid.New(), ItemName: "Fusion Coffee"})
	if err != nil {
		t.Fatalf("Analyze() error = %v, want nil", err)
	}
	if result.GrowthRatePct != nil {
		t.Fatalf("expected nil GrowthRatePct with no prior snapshot, got %v", *result.GrowthRatePct)
	}
	if result.GrowthPeriodHours != 0 {
		t.Fatalf("expected zero GrowthPeriodHours with no prior snapshot, got %v", result.GrowthPeriodHours)
	}
	if len(snapshots.saved) != 1 {
		t.Fatalf("expected exactly one snapshot saved, got %d", len(snapshots.saved))
	}
}

func TestAnalyze_GrowthCalculation(t *testing.T) {
	now := time.Now().UTC()
	previous := Snapshot{CapturedAt: now.Add(-24 * time.Hour), TotalViews: 100000}

	gen := &fakeGenerator{responses: []string{validClassification, "narrative"}}
	embedder := &fakeEmbedder{embedding: []float32{0.1}}
	searcher := &fakeSignalSearcher{signals: oneSignal(150000)}
	snapshots := &fakeSnapshotStore{recent: []Snapshot{previous}}
	agent := NewAgent(gen, embedder, searcher, snapshots, "test-model", "embed-model", 30)

	result, err := agent.Analyze(context.Background(), Input{MenuItemID: uuid.New(), ItemName: "Fusion Coffee"})
	if err != nil {
		t.Fatalf("Analyze() error = %v, want nil", err)
	}
	if result.GrowthRatePct == nil {
		t.Fatalf("expected non-nil GrowthRatePct")
	}
	const wantGrowth = 50.0 // (150000-100000)/100000 * 100
	if diff := *result.GrowthRatePct - wantGrowth; diff > 0.001 || diff < -0.001 {
		t.Fatalf("GrowthRatePct = %v, want %v", *result.GrowthRatePct, wantGrowth)
	}
	if diff := result.GrowthPeriodHours - 24.0; diff > 0.1 || diff < -0.1 {
		t.Fatalf("GrowthPeriodHours = %v, want ~24", result.GrowthPeriodHours)
	}
	if result.Summary != "narrative" {
		t.Fatalf("Summary = %q, want %q", result.Summary, "narrative")
	}
}

func TestAnalyze_ZeroBaseline(t *testing.T) {
	now := time.Now().UTC()
	previous := Snapshot{CapturedAt: now.Add(-24 * time.Hour), TotalViews: 0}

	gen := &fakeGenerator{responses: []string{validClassification, "narrative"}}
	embedder := &fakeEmbedder{embedding: []float32{0.1}}
	searcher := &fakeSignalSearcher{signals: oneSignal(5000)}
	snapshots := &fakeSnapshotStore{recent: []Snapshot{previous}}
	agent := NewAgent(gen, embedder, searcher, snapshots, "test-model", "embed-model", 30)

	result, err := agent.Analyze(context.Background(), Input{MenuItemID: uuid.New(), ItemName: "Fusion Coffee"})
	if err != nil {
		t.Fatalf("Analyze() error = %v, want nil", err)
	}
	if result.GrowthRatePct != nil {
		t.Fatalf("expected nil GrowthRatePct when previous TotalViews is 0, got %v", *result.GrowthRatePct)
	}
	if result.GrowthPeriodHours <= 0 {
		t.Fatalf("expected GrowthPeriodHours to still be populated, got %v", result.GrowthPeriodHours)
	}
}
