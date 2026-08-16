"use client"

import { Check, History, Loader2, X } from "lucide-react"

import { Badge } from "@/components/ui/badge"
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card"
import type { HashtagSuggestion } from "@/lib/api"
import { formatRelativeTime } from "./format"

const statusBadgeClassName: Record<HashtagSuggestion["status"], string> = {
  pending: "bg-secondary text-secondary-foreground",
  approved: "bg-emerald-100 text-emerald-800",
  rejected: "bg-destructive/10 text-destructive",
}

export function HashtagSuggestionHistory({
  suggestions,
  activeSuggestionId,
  onApprove,
  onReject,
  busyId,
}: {
  suggestions: HashtagSuggestion[]
  activeSuggestionId: string | null
  onApprove: (suggestion: HashtagSuggestion) => void
  onReject: (suggestion: HashtagSuggestion) => void
  busyId: string | null
}) {
  return (
    <Card className="flex-1">
      <CardHeader>
        <div className="flex items-center gap-2">
          <History className="size-4 text-muted-foreground" />
          <CardTitle className="text-base">Suggestion History</CardTitle>
        </div>
        <p className="mt-1 text-sm text-muted-foreground">
          Only an <span className="font-medium text-foreground">approved</span> suggestion is swept by
          trend-ingest — the most recently approved one wins.
        </p>
      </CardHeader>
      <CardContent className="flex flex-col gap-2.5">
        {suggestions.length === 0 ? (
          <p className="py-2 text-sm text-muted-foreground">No hashtag suggestions yet.</p>
        ) : (
          suggestions.map((suggestion) => {
            const busy = busyId === suggestion.id
            const isActive = suggestion.id === activeSuggestionId
            return (
              <div
                key={suggestion.id}
                className="flex flex-col gap-2 rounded-2xl border border-border bg-background px-3.5 py-3"
              >
                <div className="flex flex-wrap items-center justify-between gap-2">
                  <div className="flex flex-wrap items-center gap-1.5">
                    <Badge className={statusBadgeClassName[suggestion.status]}>{suggestion.status}</Badge>
                    {isActive && <Badge className="bg-primary text-primary-foreground">Active</Badge>}
                    <span className="text-xs text-muted-foreground">
                      {formatRelativeTime(suggestion.generated_at)}
                    </span>
                  </div>
                  {suggestion.status === "pending" && (
                    <div className="flex items-center gap-1.5">
                      <button
                        type="button"
                        onClick={() => onApprove(suggestion)}
                        disabled={busy}
                        aria-label="Approve suggestion"
                        className="flex items-center gap-1 rounded-full bg-emerald-100 px-2.5 py-1 text-xs font-medium text-emerald-800 hover:bg-emerald-200 disabled:pointer-events-none disabled:opacity-50"
                      >
                        {busy ? <Loader2 className="size-3.5 animate-spin" /> : <Check className="size-3.5" />}
                        Approve
                      </button>
                      <button
                        type="button"
                        onClick={() => onReject(suggestion)}
                        disabled={busy}
                        aria-label="Reject suggestion"
                        className="flex items-center gap-1 rounded-full bg-destructive/10 px-2.5 py-1 text-xs font-medium text-destructive hover:bg-destructive/20 disabled:pointer-events-none disabled:opacity-50"
                      >
                        <X className="size-3.5" />
                        Reject
                      </button>
                    </div>
                  )}
                </div>
                <div className="flex flex-wrap gap-1.5">
                  {suggestion.hashtags.map((tag) => (
                    <span
                      key={tag}
                      className="rounded-full bg-secondary/60 px-2.5 py-0.5 text-xs font-medium text-secondary-foreground"
                    >
                      {tag}
                    </span>
                  ))}
                </div>
              </div>
            )
          })
        )}
      </CardContent>
    </Card>
  )
}
