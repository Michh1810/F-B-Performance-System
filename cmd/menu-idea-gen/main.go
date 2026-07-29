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

	"github.com/joho/godotenv"

	"fbperformance/internal/agents/menuidea"
	"fbperformance/internal/config"
	"fbperformance/internal/services/llm"
	"fbperformance/internal/store"
)

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
	log.Printf("menu-idea-gen: scanning %d active menu items", len(items))

	result := menuidea.RunBatch(ctx, agent, items, ideaStore, callLogStore)

	log.Printf("menu-idea-gen: done, %d ideas generated across %d items (batch %s)", result.IdeasGenerated, result.MenuItemsScanned, result.BatchRunID)
}
