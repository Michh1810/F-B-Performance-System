"use client"

import { useEffect, useState } from "react"
import { Loader2 } from "lucide-react"

import { IngestStatusBanner } from "@/components/ingest-status-banner"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { type HashtagSuggestion, type TrendVideo, getLatestApprovedSuggestion, listHashtagSuggestions, listTrendVideos } from "@/lib/api"
import { useTrendIngestStatus } from "@/lib/use-trend-ingest-status"
import { ActiveHashtagsBanner } from "./active-hashtags-banner"
import { LastUpdatedBanner } from "./last-updated-banner"
import { TrendVideoTable } from "./trend-video-table"

const PAGE_SIZE = 50

export function TrendVideosPage() {
  const [videos, setVideos] = useState<TrendVideo[]>([])
  const [total, setTotal] = useState(0)
  const [lastUpdated, setLastUpdated] = useState<string | null>(null)
  const [loading, setLoading] = useState(true)
  const [loadingMore, setLoadingMore] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [hashtagSuggestions, setHashtagSuggestions] = useState<HashtagSuggestion[]>([])
  const [refreshToken, setRefreshToken] = useState(0)

  // Detects a running->non-running transition to auto-refresh the video
  // list once a background scrape (see IngestStatusBanner) finishes —
  // computed during render ("adjusting state when a value changes"), not in
  // an effect, matching the pattern already used in idea-review-modal.tsx
  // for pricingResetForId. Safer than an effect here since the guard
  // (status !== lastSeenIngestStatus) becomes false right after the update,
  // so it can't loop.
  const ingestStatus = useTrendIngestStatus()
  const [lastSeenIngestStatus, setLastSeenIngestStatus] = useState<string | null>(null)
  if (ingestStatus && ingestStatus.status !== lastSeenIngestStatus) {
    const wasRunning = lastSeenIngestStatus === "running"
    setLastSeenIngestStatus(ingestStatus.status)
    if (wasRunning) setRefreshToken((t) => t + 1)
  }

  // Fetch logic lives inline in the effect (not as component-level
  // functions referenced from it) so nothing runs synchronously in the
  // effect body itself — everything after the `await`/`.then` happens in a
  // microtask, which is what react-hooks/set-state-in-effect wants. `loading`
  // and `error` already start at true/null via useState, so no reset is
  // needed before the mount fetch. refreshToken re-runs this after a
  // background scrape completes, so newly-scraped videos show up without a
  // manual page reload.
  useEffect(() => {
    const controller = new AbortController()

    async function loadVideos() {
      try {
        const res = await listTrendVideos({ limit: PAGE_SIZE, offset: 0 }, controller.signal)
        setVideos(res.videos)
        setTotal(res.total)
        setLastUpdated(res.last_updated)
      } catch (err) {
        if (err instanceof DOMException && err.name === "AbortError") return
        setError(err instanceof Error ? err.message : "Failed to load scraped videos.")
      } finally {
        setLoading(false)
      }
    }

    async function loadHashtagSuggestions() {
      try {
        setHashtagSuggestions(await listHashtagSuggestions(undefined, controller.signal))
      } catch (err) {
        if (err instanceof DOMException && err.name === "AbortError") return
        // no suggestions saved yet — the banner falls back to its "none approved" state
      }
    }

    loadVideos()
    loadHashtagSuggestions()

    return () => controller.abort()
  }, [refreshToken])

  async function handleLoadMore() {
    setLoadingMore(true)
    try {
      const res = await listTrendVideos({ limit: PAGE_SIZE, offset: videos.length })
      setVideos((prev) => [...prev, ...res.videos])
      setTotal(res.total)
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to load more videos.")
    } finally {
      setLoadingMore(false)
    }
  }

  return (
    <div className="flex flex-col gap-6 pb-16">
      <header className="flex flex-wrap items-start justify-between gap-4 px-8 pt-8">
        <div>
          <div className="flex items-center gap-2.5">
            <h1 className="font-heading text-3xl font-bold text-foreground">Scraped TikTok Videos</h1>
            <Badge className="bg-secondary text-secondary-foreground">Beta</Badge>
          </div>
          <p className="mt-1.5 max-w-xl text-sm text-muted-foreground">
            The raw trend-signal corpus the Trend and Menu Idea agents search against — every video
            cmd/trend-ingest has scraped, plus any dev sample data.
          </p>
        </div>
      </header>

      <div className="flex flex-col gap-6 px-8">
        <IngestStatusBanner status={ingestStatus} />
        <ActiveHashtagsBanner activeSuggestion={getLatestApprovedSuggestion(hashtagSuggestions)} />
        <LastUpdatedBanner lastUpdated={lastUpdated} total={total} />

        <TrendVideoTable videos={videos} loading={loading} error={error} />

        {!loading && !error && videos.length < total && (
          <div className="flex justify-center">
            <Button variant="outline" className="border-border" onClick={handleLoadMore} disabled={loadingMore}>
              {loadingMore && <Loader2 className="animate-spin" />}
              Load more ({videos.length} of {total})
            </Button>
          </div>
        )}
      </div>
    </div>
  )
}
