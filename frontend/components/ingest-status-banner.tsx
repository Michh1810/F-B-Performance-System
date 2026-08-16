"use client"

import { useEffect, useState } from "react"
import { Loader2, TriangleAlert } from "lucide-react"

import { Card, CardContent } from "@/components/ui/card"
import type { TrendIngestStatus } from "@/lib/api"

function formatElapsed(startedAt: string): string {
  const seconds = Math.max(0, Math.floor((Date.now() - new Date(startedAt).getTime()) / 1000))
  const m = Math.floor(seconds / 60)
  const s = seconds % 60
  return m > 0 ? `${m}m ${s}s` : `${s}s`
}

// IngestStatusBanner is the "waiting screen" for the background TikTok
// sweep an approved hashtag suggestion triggers (see
// updateHashtagSuggestionStatus / trendingest.Service.TriggerAsync) — shown
// on both the AI Recommendations page (where Approve is clicked) and the
// Scraped Videos page (where the results land), each polling independently
// via useTrendIngestStatus.
export function IngestStatusBanner({ status }: { status: TrendIngestStatus | null }) {
  // Ticks once a second purely to re-render the elapsed-time string below —
  // a UI clock, not data fetching, so the setState here happens inside the
  // interval's callback rather than synchronously in the effect body (the
  // "subscribe to an external clock" pattern react-hooks/set-state-in-effect
  // explicitly allows).
  const [, setTick] = useState(0)
  useEffect(() => {
    if (status?.status !== "running") return
    const timer = setInterval(() => setTick((t) => t + 1), 1000)
    return () => clearInterval(timer)
  }, [status?.status])

  if (!status || status.status === "idle") return null

  if (status.status === "running") {
    return (
      <Card size="sm" className="border-primary/40 bg-primary/5">
        <CardContent className="flex flex-wrap items-center gap-3 text-sm">
          <Loader2 className="size-4 shrink-0 animate-spin text-primary" />
          <div>
            <p className="font-medium text-foreground">
              Scraping {status.hashtags.length} {status.hashtags.length === 1 ? "hashtag" : "hashtags"} for new
              TikTok videos…
            </p>
            <p className="text-xs text-muted-foreground">
              {status.started_at && `Started ${formatElapsed(status.started_at)} ago`} — this can take a few
              minutes.
            </p>
          </div>
          {status.hashtags.length > 0 && (
            <div className="ml-auto flex flex-wrap gap-1.5">
              {status.hashtags.map((tag) => (
                <span
                  key={tag}
                  className="rounded-full bg-secondary/60 px-2.5 py-0.5 text-xs font-medium text-secondary-foreground"
                >
                  {tag}
                </span>
              ))}
            </div>
          )}
        </CardContent>
      </Card>
    )
  }

  if (status.status === "failed") {
    return (
      <Card size="sm" className="border-destructive/40 bg-destructive/5">
        <CardContent className="flex items-center gap-2.5 text-sm">
          <TriangleAlert className="size-4 shrink-0 text-destructive" />
          <p className="text-destructive">Scraping TikTok videos failed: {status.error || "unknown error"}</p>
        </CardContent>
      </Card>
    )
  }

  return null
}
