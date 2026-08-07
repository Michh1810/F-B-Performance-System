"use client"

import { Eye, Flame, Sparkles, Video, XCircle } from "lucide-react"

import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { Card, CardContent } from "@/components/ui/card"
import type { MenuIdea } from "@/lib/api"
import { formatCompactNumber, formatRelativeTime } from "./format"

const EVIDENCE_PREVIEW_COUNT = 3

export function IdeaCard({
  idea,
  onReview,
  onDismiss,
  dismissing,
}: {
  idea: MenuIdea
  onReview: () => void
  onDismiss: () => void
  dismissing: boolean
}) {
  const isPromotion = idea.kind === "promotion"
  const previewVideos = idea.source_video_urls.slice(0, EVIDENCE_PREVIEW_COUNT)
  const extraVideos = idea.source_video_urls.length - previewVideos.length

  return (
    <Card>
      <CardContent className="flex flex-wrap items-start gap-5">
        <span className="flex size-20 shrink-0 items-center justify-center rounded-2xl bg-gradient-to-br from-secondary to-primary/30 text-primary-foreground/80">
          {isPromotion ? <Flame className="size-7 text-primary" /> : <Sparkles className="size-7 text-primary" />}
        </span>

        <div className="min-w-52 flex-1">
          <div className="flex flex-wrap items-center gap-2">
            <Badge className={isPromotion ? "bg-secondary text-secondary-foreground" : "bg-primary/15 text-primary"}>
              {isPromotion ? "Promotion" : "New Item Idea"}
            </Badge>
            {idea.status === "new" && <Badge className="bg-primary text-primary-foreground">NEW</Badge>}
            <span className="text-xs text-muted-foreground">{formatRelativeTime(idea.generated_at)}</span>
          </div>
          <p className="mt-1.5 font-heading text-base font-semibold text-foreground">{idea.name}</p>
          <p className="mt-1 max-w-xl text-sm leading-relaxed text-muted-foreground">{idea.rationale}</p>

          <div className="mt-3 flex flex-wrap items-center gap-1.5">
            {idea.source_hashtags.slice(0, 3).map((tag) => (
              <span key={tag} className="rounded-full bg-secondary/60 px-2.5 py-0.5 text-xs font-medium text-secondary-foreground">
                {tag}
              </span>
            ))}
            {idea.source_hashtags.length > 3 && (
              <span className="text-xs text-muted-foreground">+{idea.source_hashtags.length - 3}</span>
            )}
          </div>
          <p className="mt-2 flex items-center gap-1.5 text-xs text-muted-foreground">
            <Eye className="size-3.5" />
            {formatCompactNumber(idea.source_total_views)} total views across {idea.source_signal_count}{" "}
            {idea.source_signal_count === 1 ? "signal" : "signals"}
          </p>
        </div>

        <div className="min-w-40">
          <p className="mb-1.5 text-xs font-medium text-muted-foreground">Evidence Videos</p>
          {previewVideos.length === 0 ? (
            <p className="text-xs text-muted-foreground">No linked videos</p>
          ) : (
            <div className="flex items-center gap-1.5">
              {previewVideos.map((url) => (
                <a
                  key={url}
                  href={url}
                  target="_blank"
                  rel="noreferrer"
                  className="flex size-10 items-center justify-center rounded-xl bg-muted text-muted-foreground transition-colors hover:bg-primary/15 hover:text-primary"
                  title={url}
                >
                  <Video className="size-4" />
                </a>
              ))}
              {extraVideos > 0 && (
                <span className="flex size-10 items-center justify-center rounded-xl bg-muted text-xs font-medium text-muted-foreground">
                  +{extraVideos}
                </span>
              )}
            </div>
          )}
        </div>

        <div className="ml-auto flex shrink-0 flex-col gap-2">
          <Button size="sm" onClick={onReview}>
            Review
          </Button>
          <Button size="sm" variant="outline" className="border-border" onClick={onDismiss} disabled={dismissing}>
            <XCircle />
            Dismiss
          </Button>
        </div>
      </CardContent>
    </Card>
  )
}
