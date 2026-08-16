"use client"

import { useEffect, useState } from "react"

import { getTrendIngestStatus, type TrendIngestStatus } from "@/lib/api"

const POLL_INTERVAL_MS = 4000

// Polls GET /api/trend-ingest/status every 4s for as long as the calling
// component is mounted, so any page can show a "scraping TikTok videos in
// progress" state for the background sweep an approved hashtag suggestion
// triggers (see updateHashtagSuggestionStatus / trendingest.Service).
// Self-schedules via setTimeout (not setInterval) from inside the async
// poll function, not synchronously in the effect body — required by this
// project's react-hooks/set-state-in-effect lint rule (see
// components/trend-videos/trend-videos-page.tsx for the same pattern).
export function useTrendIngestStatus(): TrendIngestStatus | null {
  const [status, setStatus] = useState<TrendIngestStatus | null>(null)

  useEffect(() => {
    const controller = new AbortController()
    let timer: ReturnType<typeof setTimeout> | null = null

    async function poll() {
      try {
        const res = await getTrendIngestStatus(controller.signal)
        setStatus(res)
      } catch (err) {
        if (err instanceof DOMException && err.name === "AbortError") return
      } finally {
        if (!controller.signal.aborted) {
          timer = setTimeout(poll, POLL_INTERVAL_MS)
        }
      }
    }

    poll()

    return () => {
      controller.abort()
      if (timer) clearTimeout(timer)
    }
  }, [])

  return status
}
