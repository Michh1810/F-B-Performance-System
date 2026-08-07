import type { MenuIdea } from "@/lib/api"

// Derived, real numbers for the "Latest Results" strip — computed
// client-side from GET /api/ai/ideas rather than a dedicated metrics
// endpoint (none exists yet). "Latest batch" is every idea sharing the
// batch_run_id of the most-recently-generated idea.
export type LatestBatchMetrics = {
  batchRunId: string
  generatedAt: string
  ideaCount: number
  hashtagsUsed: number
  videosReferenced: number
  trendSignalsFound: number
}

export function latestBatch(ideas: MenuIdea[]): MenuIdea[] {
  if (ideas.length === 0) return []
  const latestId = ideas.reduce((latest, idea) =>
    idea.generated_at > latest.generated_at ? idea : latest
  ).batch_run_id
  return ideas.filter((idea) => idea.batch_run_id === latestId)
}

export function computeLatestBatchMetrics(ideas: MenuIdea[]): LatestBatchMetrics | null {
  const batch = latestBatch(ideas)
  if (batch.length === 0) return null

  const hashtags = new Set<string>()
  const videos = new Set<string>()
  let trendSignalsFound = 0
  let generatedAt = batch[0].generated_at

  for (const idea of batch) {
    idea.source_hashtags.forEach((tag) => hashtags.add(tag))
    idea.source_video_urls.forEach((url) => videos.add(url))
    trendSignalsFound += idea.source_signal_count
    if (idea.generated_at > generatedAt) generatedAt = idea.generated_at
  }

  return {
    batchRunId: batch[0].batch_run_id,
    generatedAt,
    ideaCount: batch.length,
    hashtagsUsed: hashtags.size,
    videosReferenced: videos.size,
    trendSignalsFound,
  }
}
