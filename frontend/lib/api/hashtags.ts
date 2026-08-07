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

// GET /api/trend-hashtags/suggestions — internal/handlers/trendhashtags.go
export function listHashtagSuggestions(
  status?: HashtagSuggestion["status"],
  signal?: AbortSignal
): Promise<HashtagSuggestion[]> {
  const query = status ? `?status=${status}` : ""
  return apiRequest<HashtagSuggestion[]>(`/api/trend-hashtags/suggestions${query}`, { signal })
}
