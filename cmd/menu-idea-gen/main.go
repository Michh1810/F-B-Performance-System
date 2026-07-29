// Command menu-idea-gen is a one-shot batch job: for each active menu item,
// it searches the trend_signals corpus for adjacent-but-not-duplicate
// TikTok content and asks the LLM to propose new item ideas, persisting any
// it finds for human review. Like cmd/trend-ingest, it is meant to be
// invoked by an external scheduler (cron, a Kubernetes CronJob, etc.) —
// this binary does not self-schedule, and it does not guard against
// overlapping invocations; that is the scheduler's responsibility.
package main

import (
	"context"
	"log"
	"sync"

	"github.com/google/uuid"
	"github.com/joho/godotenv"

	"fbperformance/internal/agents/menuidea"
	"fbperformance/internal/config"
	"fbperformance/internal/services/llm"
	"fbperformance/internal/store"
)

// ideaGenConcurrency bounds how many menu items are processed at once —
// each involves an embed call, an adjacency search, and (if there's
// evidence) a generation call.
const ideaGenConcurrency = 4

func main() {
	_ = godotenv.Load()

	cfg := config.Load()
	ctx := context.Background()

	pool, err := store.Connect(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("connect database: %v", err)
	}
	defer pool.Close()

	signalStore := store.NewTrendSignalStore(pool)
	menuItemStore := store.NewMenuItemStore(pool)
	ideaStore := store.NewMenuIdeaStore(pool)
	callLogStore := store.NewMenuIdeaCallLogStore(pool)
	llmClient := llm.NewClient(cfg.GeminiAPIKey)
	agent := menuidea.NewAgent(llmClient, llmClient, signalStore, cfg.GeminiModel, cfg.GeminiEmbedModel, cfg.TrendSignalLookbackDays)

	items, err := menuItemStore.ListActive(ctx)
	if err != nil {
		log.Fatalf("menu-idea-gen: list active menu items: %v", err)
	}

	batchRunID := uuid.New()
	log.Printf("menu-idea-gen: scanning %d active menu items (batch %s)", len(items), batchRunID)

	sem := make(chan struct{}, ideaGenConcurrency)
	var wg sync.WaitGroup
	var mu sync.Mutex
	var totalIdeas int
	for _, item := range items {
		wg.Add(1)
		go func(item menuidea.MenuItem) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()
			n := processItem(ctx, agent, ideaStore, callLogStore, batchRunID, item)
			mu.Lock()
			totalIdeas += n
			mu.Unlock()
		}(item)
	}
	wg.Wait()

	log.Printf("menu-idea-gen: done, %d ideas generated across %d items (batch %s)", totalIdeas, len(items), batchRunID)
}

// processItem generates and persists ideas for one menu item, logging and
// continuing rather than aborting the run on any error. It always persists
// a CallLog first — regardless of whether GenerateIdeas succeeded,
// degraded, or hard-failed — so every item processed in a batch run is
// inspectable afterward, including the ones that produced nothing. A hard
// failure (e.g. the LLM call erroring) still logs and moves on, but any
// candidates GenerateIdeas already produced before that failure (e.g. a
// promotion idea, which needs no LLM call) are saved regardless — the
// error only means the tweak path came up empty, not that nothing did.
func processItem(ctx context.Context, agent *menuidea.Agent, ideaStore *store.MenuIdeaStore, callLogStore *store.MenuIdeaCallLogStore, batchRunID uuid.UUID, item menuidea.MenuItem) int {
	candidates, callLog, err := agent.GenerateIdeas(ctx, item)
	if logErr := callLogStore.Save(ctx, batchRunID, callLog); logErr != nil {
		log.Printf("menu-idea-gen: item %s (%s): save call log: %v", item.Name, item.ID, logErr)
	}
	if err != nil {
		log.Printf("menu-idea-gen: item %s (%s): generate ideas: %v", item.Name, item.ID, err)
	}

	saved := 0
	for _, c := range candidates {
		idea := menuidea.StoredIdea{
			BatchRunID:             batchRunID,
			InspiredByMenuItemID:   item.ID,
			InspiredByMenuItemName: item.Name,
			Status:                 "new",
			IdeaCandidate:          c,
		}
		if err := ideaStore.Save(ctx, idea); err != nil {
			log.Printf("menu-idea-gen: item %s (%s): save idea %q: %v", item.Name, item.ID, c.Name, err)
			continue
		}
		saved++
	}
	return saved
}
