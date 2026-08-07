package menuidea

import (
	"context"
	"log"
	"sync"

	"github.com/google/uuid"
)

// batchConcurrency bounds how many menu items a single RunBatch call
// processes at once — each involves an embed call, an adjacency search, and
// (if there's evidence) a generation call.
const batchConcurrency = 4

// IdeaGenerator is the subset of Agent that RunBatch depends on, so callers
// (and tests) can supply a fake instead of a real Agent wired to live LLM
// calls.
type IdeaGenerator interface {
	GenerateIdeas(ctx context.Context, item MenuItem) ([]IdeaCandidate, CallLog, error)
}

// IdeaStore is the subset of menu-idea persistence RunBatch depends on.
type IdeaStore interface {
	Save(ctx context.Context, idea StoredIdea) error
}

// CallLogStore is the subset of call-log persistence RunBatch depends on.
type CallLogStore interface {
	Save(ctx context.Context, batchRunID uuid.UUID, log CallLog) error
}

// BatchResult summarizes one RunBatch call, for reporting back to whatever
// triggered it (a CLI log line or an HTTP response).
type BatchResult struct {
	BatchRunID       uuid.UUID
	MenuItemsScanned int
	IdeasGenerated   int
}

// RunBatch scans items concurrently, generating and persisting ideas for
// each — the shared core of cmd/menu-idea-gen (scheduler-invoked) and
// IdeasHandler.Run (POST /api/ai/ideas/run). Continues past per-item
// failures rather than aborting the batch; see processItem.
func RunBatch(ctx context.Context, agent IdeaGenerator, items []MenuItem, ideas IdeaStore, callLogs CallLogStore) BatchResult {
	batchRunID := uuid.New()

	sem := make(chan struct{}, batchConcurrency)
	var wg sync.WaitGroup
	var mu sync.Mutex
	var totalIdeas int
	for _, item := range items {
		wg.Add(1)
		go func(item MenuItem) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()
			n := processItem(ctx, agent, ideas, callLogs, batchRunID, item)
			mu.Lock()
			totalIdeas += n
			mu.Unlock()
		}(item)
	}
	wg.Wait()

	return BatchResult{
		BatchRunID:       batchRunID,
		MenuItemsScanned: len(items),
		IdeasGenerated:   totalIdeas,
	}
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
func processItem(ctx context.Context, agent IdeaGenerator, ideas IdeaStore, callLogs CallLogStore, batchRunID uuid.UUID, item MenuItem) int {
	candidates, callLog, err := agent.GenerateIdeas(ctx, item)
	if logErr := callLogs.Save(ctx, batchRunID, callLog); logErr != nil {
		log.Printf("menuidea: item %s (%s): save call log: %v", item.Name, item.ID, logErr)
	}
	if err != nil {
		log.Printf("menuidea: item %s (%s): generate ideas: %v", item.Name, item.ID, err)
	}

	saved := 0
	for _, c := range candidates {
		idea := StoredIdea{
			BatchRunID:             batchRunID,
			InspiredByMenuItemID:   item.ID,
			InspiredByMenuItemName: item.Name,
			Status:                 "new",
			IdeaCandidate:          c,
		}
		if err := ideas.Save(ctx, idea); err != nil {
			log.Printf("menuidea: item %s (%s): save idea %q: %v", item.Name, item.ID, c.Name, err)
			continue
		}
		saved++
	}
	return saved
}
