import { apiRequest } from "./client"
import type { TrendVideosResponse } from "./types"

// GET /api/v1/trend-videos — internal/handlers/trendvideos.go.
export function listTrendVideos(
  { limit, offset }: { limit?: number; offset?: number } = {},
  signal?: AbortSignal
): Promise<TrendVideosResponse> {
  const params = new URLSearchParams()
  if (limit !== undefined) params.set("limit", String(limit))
  if (offset !== undefined) params.set("offset", String(offset))
  const query = params.toString() ? `?${params.toString()}` : ""
  return apiRequest<TrendVideosResponse>(`/api/v1/trend-videos${query}`, { signal })
}
