import type { RecommendationResponse } from "@/lib/api"

export type RowState =
  | { status: "idle" }
  | { status: "loading" }
  | { status: "done"; result: RecommendationResponse; analyzedAt: string }
  | { status: "error"; error: string }
