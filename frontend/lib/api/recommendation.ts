import { apiRequest } from "./client"
import type { RecommendationResponse } from "./types"

// POST /api/ai/recommendation — internal/handlers/recommendation.go. Runs
// the Trend + Financial agents concurrently, then the Manager agent, for
// one existing menu item. Each call is a live LLM pipeline (not cached), so
// callers should treat it as a slow, one-at-a-time action per item.
export function getRecommendation(menuItemId: string, itemName: string): Promise<RecommendationResponse> {
  return apiRequest<RecommendationResponse>("/api/ai/recommendation", {
    method: "POST",
    body: { menu_item_id: menuItemId, item_name: itemName },
  })
}
