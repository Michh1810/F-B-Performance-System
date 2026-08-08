import { apiRequest } from "./client"
import type { HashtagSuggestion } from "./types"

// POST /api/trend-hashtags/suggestions — internal/handlers/trendhashtags.go.
// Grounds a fresh hashtag suggestion in the current restaurant profile +
// active menu; the profile must be saved first or the backend 422s.
export function generateHashtagSuggestion(): Promise<HashtagSuggestion> {
  return apiRequest<HashtagSuggestion>("/api/trend-hashtags/suggestions", {
    method: "POST",
  })
}

// GET /api/trend-hashtags/suggestions — internal/handlers/trendhashtags.go.
// Coerces a `null` body (Go serializes a nil slice as `null` when the table
// has zero matching rows) to `[]` so callers can always treat the result as
// an array — same reasoning as listIdeas in ideas.ts.
export async function listHashtagSuggestions(
  status?: HashtagSuggestion["status"],
  signal?: AbortSignal
): Promise<HashtagSuggestion[]> {
  const query = status ? `?status=${status}` : ""
  const suggestions = await apiRequest<HashtagSuggestion[] | null>(
    `/api/trend-hashtags/suggestions${query}`,
    { signal }
  )
  return suggestions ?? []
}

// POST /api/trend-hashtags/suggestions/manual — internal/handlers/trendhashtags.go.
// Persists a human-edited hashtag list (added/removed on the dashboard, not
// an LLM Generate call) as a new "pending" suggestion, so it goes through
// the same approve/reject review below before cmd/trend-ingest can sweep it.
export function saveManualHashtagSuggestion(hashtags: string[]): Promise<HashtagSuggestion> {
  return apiRequest<HashtagSuggestion>("/api/trend-hashtags/suggestions/manual", {
    method: "POST",
    body: { hashtags },
  })
}

// PATCH /api/trend-hashtags/suggestions/{id} — internal/handlers/trendhashtags.go.
// Approving flips this suggestion to the one cmd/trend-ingest's
// LatestApproved() will sweep next (most-recently-*reviewed* approved wins),
// and triggers a real background TikTok sweep of it — ingest_triggered is
// false when a sweep was already running (see trendingest.Service.TriggerAsync)
// or there was nothing to sweep; either way, poll getTrendIngestStatus for
// progress.
export function updateHashtagSuggestionStatus(
  id: string,
  status: Extract<HashtagSuggestion["status"], "approved" | "rejected">
): Promise<{ status: HashtagSuggestion["status"]; ingest_triggered: boolean }> {
  return apiRequest<{ status: HashtagSuggestion["status"]; ingest_triggered: boolean }>(
    `/api/trend-hashtags/suggestions/${id}`,
    {
      method: "PATCH",
      body: { status },
    }
  )
}

// getLatestApprovedSuggestion mirrors TrendHashtagSuggestionStore.LatestApproved
// (internal/store/trendhashtagsuggestions.go) — the most-recently-*reviewed*
// approved suggestion wins, since that's the hashtag list cmd/trend-ingest
// actually sweeps next. Returns null if nothing has been approved yet, in
// which case trend-ingest falls back to its built-in default hashtag list.
export function getLatestApprovedSuggestion(suggestions: HashtagSuggestion[]): HashtagSuggestion | null {
  const approved = suggestions.filter((s) => s.status === "approved")
  if (approved.length === 0) return null
  return approved.reduce((a, b) => ((a.reviewed_at ?? "") > (b.reviewed_at ?? "") ? a : b))
}
