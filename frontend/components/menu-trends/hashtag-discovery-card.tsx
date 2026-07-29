"use client"

import { useState } from "react"
import { Lightbulb, Loader2, RefreshCw, X } from "lucide-react"

import { Button } from "@/components/ui/button"
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card"
import { Input } from "@/components/ui/input"

function normalizeHashtag(raw: string): string | null {
  const trimmed = raw.trim().replace(/^#+/, "")
  if (!trimmed) return null
  return `#${trimmed.toLowerCase()}`
}

export function HashtagDiscoveryCard({
  hashtags,
  onAdd,
  onRemove,
  onGenerate,
  generating,
  canGenerate,
  error,
  disabled = false,
}: {
  hashtags: string[]
  onAdd: (tag: string) => void
  onRemove: (tag: string) => void
  onGenerate: () => void
  generating: boolean
  canGenerate: boolean
  error: string | null
  disabled?: boolean
}) {
  const [draft, setDraft] = useState("")

  function submitDraft() {
    const tag = normalizeHashtag(draft)
    if (tag && !hashtags.includes(tag)) {
      onAdd(tag)
    }
    setDraft("")
  }

  return (
    <Card className="flex-1">
      <CardHeader>
        <div className="flex items-start justify-between gap-3">
          <div>
            <CardTitle className="text-base">2. Hashtags for Trend Discovery</CardTitle>
            <p className="mt-1 text-sm text-muted-foreground">
              These hashtags will be used to discover trending TikTok videos.
            </p>
          </div>
          <Button
            variant="outline"
            size="sm"
            className="border-primary/40 text-primary hover:bg-primary/10 hover:text-primary"
            onClick={onGenerate}
            disabled={generating || !canGenerate || disabled}
            title={canGenerate ? undefined : "Add a restaurant description first"}
          >
            {generating ? <Loader2 className="animate-spin" /> : <RefreshCw />}
            Generate Hashtags
          </Button>
        </div>
      </CardHeader>
      <CardContent className="flex flex-col gap-3">
        {error && (
          <p className="rounded-xl bg-destructive/10 px-3 py-2 text-xs text-destructive">{error}</p>
        )}

        <div className="flex min-h-20 flex-wrap content-start gap-2 rounded-2xl border border-border bg-background px-3.5 py-3">
          {hashtags.length === 0 ? (
            <p className="py-1.5 text-sm text-muted-foreground">
              No hashtags yet. Generate some from your description, or add your own below.
            </p>
          ) : (
            hashtags.map((tag) => (
              <span
                key={tag}
                className="flex items-center gap-1.5 rounded-full bg-secondary px-3 py-1 text-sm font-medium text-secondary-foreground"
              >
                {tag}
                <button
                  type="button"
                  onClick={() => onRemove(tag)}
                  aria-label={`Remove ${tag}`}
                  disabled={disabled}
                  className="rounded-full text-secondary-foreground/60 hover:text-secondary-foreground disabled:pointer-events-none disabled:opacity-50"
                >
                  <X className="size-3.5" />
                </button>
              </span>
            ))
          )}
        </div>

        <div className="flex gap-2">
          <Input
            value={draft}
            onChange={(e) => setDraft(e.target.value)}
            onKeyDown={(e) => {
              if (e.key === "Enter") {
                e.preventDefault()
                submitDraft()
              }
            }}
            placeholder="Add a hashtag and press Enter"
            className="flex-1 rounded-2xl"
            disabled={disabled}
          />
          <Button variant="secondary" onClick={submitDraft} disabled={!draft.trim() || disabled}>
            Add
          </Button>
        </div>

        <p className="flex items-center gap-1.5 text-xs text-muted-foreground">
          <Lightbulb className="size-3.5" />
          Add more hashtags to improve trend discovery
        </p>
      </CardContent>
    </Card>
  )
}
