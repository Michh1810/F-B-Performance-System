"use client"

import { ExternalLink, Eye, Heart, MessageCircle, Share2, Video } from "lucide-react"

import { Badge } from "@/components/ui/badge"
import { Card, CardContent } from "@/components/ui/card"
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table"
import type { TrendVideo } from "@/lib/api"
import { isMockVideoUrl } from "@/lib/mock-video-url"
import { formatCompactNumber, formatRelativeTime } from "./format"

const columnHeadClassName = "text-xs font-semibold tracking-wide text-primary uppercase"

export function TrendVideoTable({
  videos,
  loading,
  error,
}: {
  videos: TrendVideo[]
  loading: boolean
  error: string | null
}) {
  if (loading) {
    return (
      <Card>
        <CardContent className="h-40 animate-pulse bg-muted/50" />
      </Card>
    )
  }

  if (error) {
    return (
      <Card>
        <CardContent className="py-8 text-center text-sm text-destructive">{error}</CardContent>
      </Card>
    )
  }

  if (videos.length === 0) {
    return (
      <Card>
        <CardContent className="flex flex-col items-center gap-2 py-10 text-center">
          <Video className="size-6 text-muted-foreground" />
          <p className="text-sm font-medium text-foreground">No videos scraped yet</p>
          <p className="text-sm text-muted-foreground">
            Run <code className="text-xs">go run ./cmd/trend-ingest</code> to sweep real TikTok content.
          </p>
        </CardContent>
      </Card>
    )
  }

  return (
    <Card size="sm" className="overflow-hidden">
      <Table>
        <TableHeader>
          <TableRow>
            <TableHead className={columnHeadClassName}>Video</TableHead>
            <TableHead className={columnHeadClassName}>Source</TableHead>
            <TableHead className={columnHeadClassName}>Hashtags</TableHead>
            <TableHead className={`${columnHeadClassName} text-right`}>Engagement</TableHead>
            <TableHead className={columnHeadClassName}>Posted</TableHead>
            <TableHead className={columnHeadClassName}>Scraped</TableHead>
          </TableRow>
        </TableHeader>
        <TableBody>
          {videos.map((video) => {
            const isMock = isMockVideoUrl(video.url)
            return (
              <TableRow key={video.id}>
                <TableCell className="max-w-xs">
                  <div className="flex items-start gap-2">
                    {isMock || !video.url ? (
                      <span
                        className="mt-0.5 flex size-6 shrink-0 items-center justify-center rounded-lg bg-muted text-muted-foreground/50"
                        title={isMock ? "Sample data — not a real TikTok video" : undefined}
                      >
                        <Video className="size-3.5" />
                      </span>
                    ) : (
                      <a
                        href={video.url}
                        target="_blank"
                        rel="noreferrer"
                        className="mt-0.5 flex size-6 shrink-0 items-center justify-center rounded-lg bg-muted text-muted-foreground transition-colors hover:bg-primary/15 hover:text-primary"
                        title={video.url}
                      >
                        <ExternalLink className="size-3.5" />
                      </a>
                    )}
                    <p className="line-clamp-2 text-sm text-foreground">{video.caption}</p>
                  </div>
                </TableCell>
                <TableCell>
                  <Badge variant="outline" className="border-border capitalize text-muted-foreground">
                    {video.source}
                  </Badge>
                </TableCell>
                <TableCell className="max-w-40">
                  <div className="flex flex-wrap gap-1">
                    {video.hashtags.slice(0, 3).map((tag) => (
                      <span
                        key={tag}
                        className="rounded-full bg-secondary/60 px-2 py-0.5 text-xs font-medium text-secondary-foreground"
                      >
                        {tag}
                      </span>
                    ))}
                    {video.hashtags.length > 3 && (
                      <span className="text-xs text-muted-foreground">+{video.hashtags.length - 3}</span>
                    )}
                  </div>
                </TableCell>
                <TableCell>
                  <div className="flex items-center justify-end gap-3 text-xs text-muted-foreground">
                    <span className="flex items-center gap-1" title="Views">
                      <Eye className="size-3.5" />
                      {formatCompactNumber(video.view_count)}
                    </span>
                    <span className="flex items-center gap-1" title="Likes">
                      <Heart className="size-3.5" />
                      {formatCompactNumber(video.like_count)}
                    </span>
                    <span className="flex items-center gap-1" title="Comments">
                      <MessageCircle className="size-3.5" />
                      {formatCompactNumber(video.comment_count)}
                    </span>
                    <span className="flex items-center gap-1" title="Shares">
                      <Share2 className="size-3.5" />
                      {formatCompactNumber(video.share_count)}
                    </span>
                  </div>
                </TableCell>
                <TableCell className="text-xs text-muted-foreground">
                  {video.posted_at ? formatRelativeTime(video.posted_at) : "—"}
                </TableCell>
                <TableCell className="text-xs text-muted-foreground">{formatRelativeTime(video.ingested_at)}</TableCell>
              </TableRow>
            )
          })}
        </TableBody>
      </Table>
    </Card>
  )
}
