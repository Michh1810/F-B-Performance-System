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
  price_cents: number
  cogs_cents: number
  // has_sufficient_history is false for a never-sold or too-recently-added
  // item — the Financial Agent's forecast needs real sales history to be
  // meaningful, so the Menu Strategy table hides "Analyze" until this flips.
  has_sufficient_history: boolean
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

// RunSummary is what POST /api/ai/ideas/run reports back — see runs.ts.
export type RunSummary = {
  batch_run_id: string
  started_at: string
  completed_at: string
  menu_items_scanned: number
  ideas_generated: number
}

// Decision mirrors manager.Decision — the Manager Agent's only three
// possible calls on an existing menu item. There is no "IMPROVE" — LAUNCH
// is reused to mean "keep it, it's working" for an existing item (see
// internal/agents/manager/prompt.go).
export type Decision = "LAUNCH" | "CUT" | "REPRICE"

// TrendMetrics mirrors orchestrator.TrendMetrics — the structured TikTok
// signal the Trend Agent grounded its narrative in.
export type TrendMetrics = {
  video_count: number
  total_views: number
  total_likes: number
  total_comments: number
  total_shares: number
  engagement_rate: number
  sentiment_label: string
  sentiment_score: number
  growth_rate_pct: number | null
  growth_period_hours: number
  top_hashtags: string[]
  related_trends: string[]
}

// RecommendationResponse mirrors orchestrator.Response — POST
// /api/ai/recommendation's body, synthesized from the Trend, Financial, and
// Manager agents running concurrently then converging on one decision.
export type RecommendationResponse = {
  item_name: string
  decision: Decision
  reasoning: string
  trend_analysis: string
  financial_analysis: string
  trend_metrics: TrendMetrics
}
