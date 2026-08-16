import { apiRequest } from "./client"
import type { TrendIngestStatus } from "./types"

// GET /api/trend-ingest/status — internal/handlers/trendingest.go. Reports
// the single background TikTok sweep trendingest.Service tracks, kicked off
// by approving a hashtag suggestion (see updateHashtagSuggestionStatus).
export function getTrendIngestStatus(signal?: AbortSignal): Promise<TrendIngestStatus> {
  return apiRequest<TrendIngestStatus>("/api/trend-ingest/status", { signal })
}
