"use client"

import { Loader2, RefreshCw, ThumbsDown, ThumbsUp, TrendingUp } from "lucide-react"

import { Button } from "@/components/ui/button"
import { Card, CardContent } from "@/components/ui/card"
import type { ActiveMenuItem, RecommendationResponse } from "@/lib/api"
import { DecisionBadge } from "./decision-badge"
import { formatCents, formatCompactNumber, formatPercent, formatRelativeTime } from "./format"
import { MarkdownText } from "./markdown-text"

export function RecommendationDetail({
  item,
  result,
  analyzedAt,
  analyzing,
  onReanalyze,
}: {
  item: ActiveMenuItem
  result: RecommendationResponse
  analyzedAt: string
  analyzing: boolean
  onReanalyze: () => void
}) {
  const metrics = result.trend_metrics

  return (
    <div className="flex flex-col gap-5">
      <div className="flex flex-wrap items-start justify-between gap-4">
        <div>
          <div className="flex items-center gap-2.5">
            <h1 className="font-heading text-2xl font-bold text-foreground">{item.name}</h1>
            <DecisionBadge decision={result.decision} />
          </div>
          <p className="mt-1 text-sm text-muted-foreground">
            {item.category} · {formatCents(item.price_cents)}
          </p>
          <p className="mt-1 text-xs text-muted-foreground" title={analyzedAt}>
            Updated {formatRelativeTime(analyzedAt)}
          </p>
        </div>
        <Button onClick={onReanalyze} disabled={analyzing}>
          {analyzing ? <Loader2 className="animate-spin" /> : <RefreshCw />}
          Re-analyze
        </Button>
      </div>

      <Card>
        <CardContent>
          <p className="text-xs font-medium tracking-wide text-muted-foreground uppercase">
            Manager Agent Reasoning
          </p>
          <MarkdownText className="mt-1.5 text-foreground/90">{result.reasoning}</MarkdownText>
        </CardContent>
      </Card>

      <Card>
        <CardContent>
          <p className="flex items-center gap-1.5 text-sm font-medium text-foreground">
            <TrendingUp className="size-4 text-primary" />
            Trend Analysis
          </p>
          <MarkdownText className="mt-1.5">{result.trend_analysis}</MarkdownText>

          <dl className="mt-4 grid grid-cols-2 gap-x-4 gap-y-3 border-t border-border pt-4 text-xs sm:grid-cols-3">
            <Metric label="Videos" value={metrics.video_count.toLocaleString()} />
            <Metric label="Total Views" value={formatCompactNumber(metrics.total_views)} />
            <Metric label="Engagement" value={formatPercent(metrics.engagement_rate * 100)} />
            <Metric
              label="Growth"
              value={
                metrics.growth_rate_pct === null
                  ? "No baseline yet"
                  : `${formatPercent(metrics.growth_rate_pct, { signed: true })} / ${metrics.growth_period_hours.toFixed(0)}h`
              }
            />
            <Metric label="Sentiment" value={`${metrics.sentiment_label} (${metrics.sentiment_score.toFixed(2)})`} />
          </dl>

          {metrics.top_hashtags.length > 0 && (
            <div className="mt-4 flex flex-wrap gap-1.5 border-t border-border pt-4">
              {metrics.top_hashtags.map((tag) => (
                <span key={tag} className="rounded-full bg-secondary/60 px-2.5 py-0.5 text-xs font-medium text-secondary-foreground">
                  {tag}
                </span>
              ))}
            </div>
          )}
        </CardContent>
      </Card>

      <Card>
        <CardContent>
          <p className="flex items-center gap-1.5 text-sm font-medium text-foreground">
            {result.decision === "CUT" ? (
              <ThumbsDown className="size-4 text-destructive" />
            ) : (
              <ThumbsUp className="size-4 text-primary" />
            )}
            Financial Analysis
          </p>
          <MarkdownText className="mt-1.5">{result.financial_analysis}</MarkdownText>
        </CardContent>
      </Card>
    </div>
  )
}

function Metric({ label, value }: { label: string; value: string }) {
  return (
    <div>
      <dt className="text-muted-foreground">{label}</dt>
      <dd className="font-medium text-foreground">{value}</dd>
    </div>
  )
}
