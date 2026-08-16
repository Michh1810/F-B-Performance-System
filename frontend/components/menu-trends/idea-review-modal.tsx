"use client"

import { useState } from "react"
import { AlertTriangle, CheckCircle2, ExternalLink, Loader2 } from "lucide-react"

import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import type { MenuIdea } from "@/lib/api"
import { isMockVideoUrl } from "@/lib/mock-video-url"
import { formatCompactNumber } from "./format"

export function IdeaReviewModal({
  idea,
  open,
  onOpenChange,
  onDismiss,
  onPromote,
  submitting,
  error,
}: {
  idea: MenuIdea | null
  open: boolean
  onOpenChange: (open: boolean) => void
  onDismiss: (idea: MenuIdea) => void
  onPromote: (idea: MenuIdea, pricing?: { priceCents: number; cogsCents: number }) => void
  submitting: boolean
  error: string | null
}) {
  const [price, setPrice] = useState("")
  const [cogs, setCogs] = useState("")
  // Reset the pricing draft when a different idea is opened for review —
  // done during render (not an effect) per React's "adjusting state when a
  // prop changes" pattern, since it must happen before the form paints.
  const [pricingResetForId, setPricingResetForId] = useState<string | null>(null)
  if (idea && idea.id !== pricingResetForId) {
    setPricingResetForId(idea.id)
    setPrice("")
    setCogs("")
  }

  if (!idea) return null

  const isPromotion = idea.kind === "promotion"
  const alreadyPromoted = idea.status === "promoted"
  const priceValue = Number(price)
  const cogsValue = Number(cogs)
  const pricingValid = price !== "" && cogs !== "" && priceValue > 0 && cogsValue >= 0

  function handleApprove() {
    if (!idea) return
    if (isPromotion) {
      onPromote(idea)
      return
    }
    if (!pricingValid) return
    onPromote(idea, {
      priceCents: Math.round(priceValue * 100),
      cogsCents: Math.round(cogsValue * 100),
    })
  }

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-xl">
        <DialogHeader>
          <div className="flex items-center gap-2">
            <Badge className={isPromotion ? "bg-secondary text-secondary-foreground" : "bg-primary/15 text-primary"}>
              {isPromotion ? "Promotion" : "New Item Idea"}
            </Badge>
            <Badge variant="outline" className="border-border capitalize text-muted-foreground">
              {idea.status}
            </Badge>
          </div>
          <DialogTitle>{idea.name}</DialogTitle>
          {idea.inspired_by_menu_item_name && (
            <DialogDescription>
              Inspired by existing item: <span className="font-medium text-foreground">{idea.inspired_by_menu_item_name}</span>
            </DialogDescription>
          )}
        </DialogHeader>

        <div className="flex flex-col gap-4">
          <div>
            <p className="text-xs font-medium tracking-wide text-muted-foreground uppercase">AI Explanation</p>
            <p className="mt-1 text-sm leading-relaxed text-foreground">{idea.rationale}</p>
            {idea.description && idea.description !== idea.rationale && (
              <p className="mt-1 text-sm leading-relaxed text-muted-foreground">{idea.description}</p>
            )}
          </div>

          <div className="grid grid-cols-2 gap-4 text-sm">
            <div>
              <p className="text-xs font-medium tracking-wide text-muted-foreground uppercase">Hashtags</p>
              <div className="mt-1.5 flex flex-wrap gap-1.5">
                {idea.source_hashtags.length === 0 ? (
                  <span className="text-muted-foreground">None recorded</span>
                ) : (
                  idea.source_hashtags.map((tag) => (
                    <span key={tag} className="rounded-full bg-secondary/60 px-2.5 py-0.5 text-xs font-medium text-secondary-foreground">
                      {tag}
                    </span>
                  ))
                )}
              </div>
            </div>
            <div>
              <p className="text-xs font-medium tracking-wide text-muted-foreground uppercase">View Counts</p>
              <p className="mt-1.5 text-foreground">
                {formatCompactNumber(idea.source_total_views)} total views · {idea.source_signal_count}{" "}
                {idea.source_signal_count === 1 ? "signal" : "signals"}
              </p>
            </div>
          </div>

          <div>
            <p className="text-xs font-medium tracking-wide text-muted-foreground uppercase">Evidence Videos</p>
            {idea.source_video_urls.length === 0 ? (
              <p className="mt-1.5 text-sm text-muted-foreground">No linked videos</p>
            ) : (
              <ul className="mt-1.5 flex flex-col gap-1">
                {idea.source_video_urls.map((url, i) =>
                  isMockVideoUrl(url) ? (
                    <li
                      key={url}
                      className="flex items-center gap-1.5 text-sm text-muted-foreground"
                      title="Sample data from cmd/seed-idea-trends — not a real TikTok video"
                    >
                      <ExternalLink className="size-3.5 shrink-0" />
                      Evidence video {i + 1} (sample data, not clickable)
                    </li>
                  ) : (
                    <li key={url}>
                      <a
                        href={url}
                        target="_blank"
                        rel="noreferrer"
                        className="flex items-center gap-1.5 text-sm text-primary hover:underline"
                      >
                        <ExternalLink className="size-3.5 shrink-0" />
                        Evidence video {i + 1}
                      </a>
                    </li>
                  )
                )}
              </ul>
            )}
          </div>

          {alreadyPromoted && (
            <div className="flex items-start gap-2 rounded-2xl bg-emerald-50 px-3.5 py-3 text-sm text-emerald-900">
              <CheckCircle2 className="mt-0.5 size-4 shrink-0" />
              <span>
                This idea has already been promoted.
                {idea.created_menu_item_id && " A menu item was created from it."}
              </span>
            </div>
          )}

          {!isPromotion && !alreadyPromoted && (
            <div className="rounded-2xl border border-border p-4">
              <p className="text-sm font-medium text-foreground">Pricing (required to approve)</p>
              <p className="mt-0.5 text-xs text-muted-foreground">
                The AI never sets pricing — enter the price and cost of goods sold yourself.
              </p>
              <div className="mt-3 grid grid-cols-2 gap-3">
                <div>
                  <Label htmlFor="idea-price">Price ($)</Label>
                  <Input
                    id="idea-price"
                    type="number"
                    min="0"
                    step="0.01"
                    value={price}
                    onChange={(e) => setPrice(e.target.value)}
                    placeholder="12.99"
                    className="mt-1"
                  />
                </div>
                <div>
                  <Label htmlFor="idea-cogs">COGS ($)</Label>
                  <Input
                    id="idea-cogs"
                    type="number"
                    min="0"
                    step="0.01"
                    value={cogs}
                    onChange={(e) => setCogs(e.target.value)}
                    placeholder="4.50"
                    className="mt-1"
                  />
                </div>
              </div>
            </div>
          )}

          {error && (
            <p className="flex items-center gap-1.5 rounded-xl bg-destructive/10 px-3 py-2 text-xs text-destructive">
              <AlertTriangle className="size-3.5 shrink-0" />
              {error}
            </p>
          )}
        </div>

        <DialogFooter>
          <Button variant="outline" className="border-border" onClick={() => onDismiss(idea)} disabled={submitting || alreadyPromoted}>
            Dismiss
          </Button>
          <Button
            onClick={handleApprove}
            disabled={submitting || alreadyPromoted || (!isPromotion && !pricingValid)}
          >
            {submitting && <Loader2 className="animate-spin" />}
            {isPromotion ? "Approve / Promote" : "Approve & Create Menu Item"}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}
