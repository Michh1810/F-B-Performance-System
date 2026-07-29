// Wire types for the Menu Trends & Ideas page. Field names/shapes mirror
// the Go JSON tags exactly (see internal/store, internal/agents/menuidea) —
// this file has no independent source of truth.

export type RestaurantProfile = {
  description: string
  updated_at: string | null
}

export type HashtagSuggestion = {
  id: string
  source_description: string
  hashtags: string[]
  status: "pending" | "approved" | "rejected"
  generated_at: string
  reviewed_at: string | null
}

export type ActiveMenuItem = {
  id: string
  name: string
  category: string
}

// IdeaKind mirrors menuidea.IdeaCandidate.Kind: "promotion" is a
// deterministic call to feature an existing item; "tweak" is an
// LLM-proposed new item.
export type IdeaKind = "tweak" | "promotion"

export type IdeaStatus = "new" | "reviewed" | "dismissed" | "promoted"

export type MenuIdea = {
  id: string
  batch_run_id: string
  inspired_by_menu_item_id: string
  inspired_by_menu_item_name: string
  status: IdeaStatus
  generated_at: string
  reviewed_at: string | null
  created_menu_item_id: string | null

  kind: IdeaKind
  name: string
  description: string
  suggested_category: string
  rationale: string
  source_hashtags: string[]
  source_signal_count: number
  source_total_views: number
  source_video_urls: string[]
}

export type UpdateIdeaStatusRequest = {
  status: IdeaStatus
  price_cents?: number
  cogs_cents?: number
}

// RunSummary is what the Menu Idea Agent run CTA reports back — see
// runMenuIdeaAgent's PLACEHOLDER_ENDPOINT note in ideas.ts for why this
// isn't backed by a real endpoint yet.
export type RunSummary = {
  batch_run_id: string
  started_at: string
  completed_at: string
  menu_items_scanned: number
  ideas_generated: number
}
