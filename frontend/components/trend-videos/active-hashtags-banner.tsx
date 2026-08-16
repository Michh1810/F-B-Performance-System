"use client"

import { Hash, TriangleAlert } from "lucide-react"

import { Card, CardContent } from "@/components/ui/card"
import type { HashtagSuggestion } from "@/lib/api"

export function ActiveHashtagsBanner({ activeSuggestion }: { activeSuggestion: HashtagSuggestion | null }) {
  return (
    <Card size="sm">
      <CardContent className="flex flex-wrap items-center gap-2.5 text-sm">
        {activeSuggestion ? (
          <>
            <Hash className="size-4 shrink-0 text-muted-foreground" />
            <span className="font-medium text-foreground">Hashtags currently in use:</span>
            <div className="flex flex-wrap gap-1.5">
              {activeSuggestion.hashtags.map((tag) => (
                <span
                  key={tag}
                  className="rounded-full bg-secondary/60 px-2.5 py-0.5 text-xs font-medium text-secondary-foreground"
                >
                  {tag}
                </span>
              ))}
            </div>
          </>
        ) : (
          <>
            <TriangleAlert className="size-4 shrink-0 text-muted-foreground" />
            <span className="text-muted-foreground">
              No hashtags have been approved yet — trend-ingest is sweeping its built-in default list
              instead of anything tailored to your menu.
            </span>
          </>
        )}
      </CardContent>
    </Card>
  )
}
