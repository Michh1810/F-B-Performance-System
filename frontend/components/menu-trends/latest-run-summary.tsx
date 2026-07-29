"use client"

import { FileText, Hash, Lightbulb, ScanLine, Video, Waves } from "lucide-react"

import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { Card, CardContent } from "@/components/ui/card"
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip"
import type { LatestBatchMetrics } from "@/lib/idea-metrics"

function formatTimestamp(iso: string) {
  return new Date(iso).toLocaleString(undefined, {
    month: "short",
    day: "numeric",
    year: "numeric",
    hour: "numeric",
    minute: "2-digit",
  })
}

export function LatestRunSummary({
  metrics,
  activeMenuItemsCount,
}: {
  metrics: LatestBatchMetrics | null
  activeMenuItemsCount: number
}) {
  return (
    <div className="flex flex-col gap-4">
      <div className="flex flex-wrap items-center justify-between gap-3">
        <div className="flex flex-wrap items-center gap-3">
          <h2 className="font-heading text-lg font-semibold text-foreground">Latest Results</h2>
          {metrics && (
            <>
              <span className="text-sm text-muted-foreground">{formatTimestamp(metrics.generatedAt)}</span>
              <Badge className="bg-emerald-100 text-emerald-800">Completed</Badge>
            </>
          )}
        </div>

        <Tooltip>
          <TooltipTrigger asChild>
            <span>
              <Button variant="outline" size="sm" className="border-border bg-card" disabled>
                <FileText />
                View Full Run Log
              </Button>
            </span>
          </TooltipTrigger>
          <TooltipContent>Per-item run logs aren&apos;t available in the UI yet.</TooltipContent>
        </Tooltip>
      </div>

      {!metrics ? (
        <Card>
          <CardContent className="py-8 text-center text-sm text-muted-foreground">
            No runs yet. Click &quot;Run Menu Idea Agent&quot; above to generate your first ideas.
          </CardContent>
        </Card>
      ) : (
        <div className="grid grid-cols-2 gap-4 sm:grid-cols-3 lg:grid-cols-5">
          <MetricCard icon={Hash} label="Hashtags Used" value={metrics.hashtagsUsed} />
          <MetricCard icon={Video} label="TikTok Videos Referenced" value={metrics.videosReferenced} />
          <MetricCard icon={Waves} label="Trend Signals Found" value={metrics.trendSignalsFound} />
          <MetricCard icon={Lightbulb} label="Ideas Generated" value={metrics.ideaCount} />
          <MetricCard icon={ScanLine} label="Menu Items Scanned" value={activeMenuItemsCount} />
        </div>
      )}
    </div>
  )
}

function MetricCard({
  icon: Icon,
  label,
  value,
}: {
  icon: React.ComponentType<{ className?: string }>
  label: string
  value: number
}) {
  return (
    <Card size="sm">
      <CardContent className="flex items-center gap-3">
        <span className="flex size-9 shrink-0 items-center justify-center rounded-xl bg-secondary text-secondary-foreground">
          <Icon className="size-4.5" />
        </span>
        <div>
          <p className="font-heading text-xl font-bold text-foreground">{value.toLocaleString()}</p>
          <p className="text-xs text-muted-foreground">{label}</p>
        </div>
      </CardContent>
    </Card>
  )
}
