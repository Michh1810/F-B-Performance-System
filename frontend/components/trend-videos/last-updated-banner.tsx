"use client"

import { Clock } from "lucide-react"

import { Card, CardContent } from "@/components/ui/card"
import { formatDateTime, formatRelativeTime } from "./format"

export function LastUpdatedBanner({
  lastUpdated,
  total,
}: {
  lastUpdated: string | null
  total: number
}) {
  return (
    <Card size="sm">
      <CardContent className="flex flex-wrap items-center gap-2.5 text-sm">
        <Clock className="size-4 text-muted-foreground" />
        {lastUpdated ? (
          <>
            <span className="font-medium text-foreground">Last updated {formatDateTime(lastUpdated)}</span>
            <span className="text-muted-foreground">({formatRelativeTime(lastUpdated)})</span>
          </>
        ) : (
          <span className="text-muted-foreground">No videos scraped yet</span>
        )}
        <span className="ml-auto text-muted-foreground">
          {total} {total === 1 ? "video" : "videos"} in corpus
        </span>
      </CardContent>
    </Card>
  )
}
