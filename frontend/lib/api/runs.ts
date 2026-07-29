import { apiRequest } from "./client"
import type { RunSummary } from "./types"

// PLACEHOLDER ENDPOINT — no backend route exists for this yet.
//
// Today the Menu Idea Agent only runs as a scheduler-invoked batch binary
// (cmd/menu-idea-gen) and hashtag-driven TikTok ingestion is a separate CLI
// (cmd/trend-ingest); neither is wired to HTTP. This function is the seam
// the "Run Menu Idea Agent" button calls through, so wiring up the real
// endpoint later only means implementing this request server-side — no
// frontend changes. Until then it will 404, which callers should treat as
// "not implemented yet" (see ApiError.notImplemented in client.ts) rather
// than an inline fetch failure.
export function runMenuIdeaAgent(): Promise<RunSummary> {
  return apiRequest<RunSummary>("/api/ai/ideas/run", { method: "POST" })
}
