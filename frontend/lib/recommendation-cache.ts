import type { ActiveMenuItem, RecommendationResponse } from "@/lib/api"

// Recommendation results only ever live in the POST /api/ai/recommendation
// response — there's no backend persistence for them (unlike menu_ideas).
// Since the detail view is now its own route, navigating to it unmounts the
// table page's in-memory state, so results are cached here (sessionStorage,
// not localStorage — these are live LLM analyses, not records worth
// keeping past the tab closing) keyed by menu item id. Both the table (to
// show "Done" after a back-navigation) and the detail page (to render
// without re-running the analysis, and to survive a refresh) read from this.
const KEY_PREFIX = "menu-strategy:recommendation:"

export type CachedRecommendation = {
  item: ActiveMenuItem
  result: RecommendationResponse
  analyzedAt: string
}

export function cacheRecommendation(item: ActiveMenuItem, result: RecommendationResponse): CachedRecommendation {
  const data: CachedRecommendation = { item, result, analyzedAt: new Date().toISOString() }
  if (typeof window !== "undefined") {
    window.sessionStorage.setItem(KEY_PREFIX + item.id, JSON.stringify(data))
  }
  return data
}

export function getCachedRecommendation(itemId: string): CachedRecommendation | null {
  if (typeof window === "undefined") return null
  const raw = window.sessionStorage.getItem(KEY_PREFIX + itemId)
  if (!raw) return null
  try {
    return JSON.parse(raw) as CachedRecommendation
  } catch {
    return null
  }
}

// listCachedRecommendationIds is used to hydrate the table's "Done" status
// on mount, without pulling every cached result's full payload into memory.
export function listCachedRecommendationIds(): string[] {
  if (typeof window === "undefined") return []
  const ids: string[] = []
  for (let i = 0; i < window.sessionStorage.length; i++) {
    const key = window.sessionStorage.key(i)
    if (key?.startsWith(KEY_PREFIX)) {
      ids.push(key.slice(KEY_PREFIX.length))
    }
  }
  return ids
}

// REANALYZE_FRESHNESS_DAYS gates the "re-analyze" confirmation prompt: an
// analysis younger than this is unlikely to have moved much (trend signals
// and financial forecasts don't shift meaningfully day to day), so
// re-running it just spends another LLM call for a near-identical result.
export const REANALYZE_FRESHNESS_DAYS = 10

export function daysSince(iso: string): number {
  const elapsedMs = Date.now() - new Date(iso).getTime()
  return Math.floor(elapsedMs / (24 * 60 * 60 * 1000))
}
