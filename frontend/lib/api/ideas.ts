import { apiRequest } from "./client"
import type { IdeaStatus, MenuIdea } from "./types"

// GET /api/ai/ideas — internal/handlers/ideas.go. Coerces a `null` body
// (Go serializes a nil slice as `null` when there are zero matching rows)
// to `[]` so callers can always treat the result as an array.
export async function listIdeas(status?: IdeaStatus, signal?: AbortSignal): Promise<MenuIdea[]> {
  const query = status ? `?status=${status}` : ""
  const ideas = await apiRequest<MenuIdea[] | null>(`/api/ai/ideas${query}`, { signal })
  return ideas ?? []
}

// PATCH /api/ai/ideas/{id} with status "reviewed" | "dismissed" | "new" —
// internal/handlers/ideas.go. Use promoteIdea (not this) for "promoted",
// since that transition can create a real menu_items row.
export function setIdeaStatus(
  id: string,
  status: Exclude<IdeaStatus, "promoted">
): Promise<{ status: IdeaStatus }> {
  return apiRequest<{ status: IdeaStatus }>(`/api/ai/ideas/${id}`, {
    method: "PATCH",
    body: { status },
  })
}

// PATCH /api/ai/ideas/{id} with status "promoted" — internal/handlers/ideas.go.
// For "promotion"-kind ideas this only flips status. For "tweak"-kind ideas
// it creates a real menu_items row priced by priceCents/cogsCents, which the
// AI never sets — a human must supply both. Idempotent on the backend: a
// second call against an already-promoted idea returns it unchanged rather
// than creating a duplicate menu item.
export function promoteIdea(
  id: string,
  pricing?: { priceCents: number; cogsCents: number }
): Promise<MenuIdea> {
  return apiRequest<MenuIdea>(`/api/ai/ideas/${id}`, {
    method: "PATCH",
    body: {
      status: "promoted",
      price_cents: pricing?.priceCents ?? 0,
      cogs_cents: pricing?.cogsCents ?? 0,
    },
  })
}
