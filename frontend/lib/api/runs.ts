import { apiRequest } from "./client"
import type { RunSummary } from "./types"

// POST /api/ai/ideas/run — internal/handlers/ideas.go. Runs the same
// scan-active-items-and-generate-ideas pass as the scheduler-invoked
// cmd/menu-idea-gen, synchronously, against whatever trend signals are
// already in the corpus (hashtag-driven TikTok ingestion itself is still a
// separate CLI, cmd/trend-ingest, not triggered by this call).
export function runMenuIdeaAgent(): Promise<RunSummary> {
  return apiRequest<RunSummary>("/api/ai/ideas/run", { method: "POST" })
}
