"use client"

import { useEffect, useState } from "react"
import Link from "next/link"
import { useParams } from "next/navigation"
import { AlertTriangle, ArrowLeft, Loader2, Sparkles } from "lucide-react"

import { Button } from "@/components/ui/button"
import { Card, CardContent } from "@/components/ui/card"
import { RecommendationDetail } from "@/components/menu-strategy/recommendation-detail"
import { ReanalyzeConfirmDialog } from "@/components/menu-strategy/reanalyze-confirm-dialog"
import { useReanalyzeGate } from "@/components/menu-strategy/use-reanalyze-gate"
import { formatCents } from "@/components/menu-strategy/format"
import { getRecommendation, listActiveMenuItems, type ActiveMenuItem, type RecommendationResponse } from "@/lib/api"
import { cacheRecommendation, getCachedRecommendation } from "@/lib/recommendation-cache"

type PageState =
  | { status: "loading" }
  | { status: "not-found" }
  | { status: "needs-analysis"; item: ActiveMenuItem }
  | { status: "analyzing"; item: ActiveMenuItem }
  | { status: "ready"; item: ActiveMenuItem; result: RecommendationResponse; analyzedAt: string }
  | { status: "error"; item: ActiveMenuItem; message: string }

export default function MenuItemRecommendationPage() {
  const params = useParams<{ itemId: string }>()
  const itemId = params.itemId

  const [state, setState] = useState<PageState>(() => {
    const cached = getCachedRecommendation(itemId)
    return cached
      ? { status: "ready", item: cached.item, result: cached.result, analyzedAt: cached.analyzedAt }
      : { status: "loading" }
  })
  // Re-check the cache during render (not in the effect below) whenever the
  // route param itself changes — React's "adjusting state when a prop
  // changes" pattern, same as idea-review-modal.tsx's pricing-draft reset.
  const [checkedItemId, setCheckedItemId] = useState(itemId)
  if (itemId !== checkedItemId) {
    setCheckedItemId(itemId)
    const cached = getCachedRecommendation(itemId)
    setState(
      cached
        ? { status: "ready", item: cached.item, result: cached.result, analyzedAt: cached.analyzedAt }
        : { status: "loading" }
    )
  }

  useEffect(() => {
    // Already resolved from cache during render above — nothing to fetch.
    if (getCachedRecommendation(itemId)) return

    const controller = new AbortController()
    listActiveMenuItems(controller.signal)
      .then((items) => {
        const item = items.find((i) => i.id === itemId)
        setState(item ? { status: "needs-analysis", item } : { status: "not-found" })
      })
      .catch((err: unknown) => {
        if (err instanceof DOMException && err.name === "AbortError") return
        setState({ status: "not-found" })
      })
    return () => controller.abort()
  }, [itemId])

  async function runAnalysis(item: ActiveMenuItem) {
    setState({ status: "analyzing", item })
    try {
      const result = await getRecommendation(item.id, item.name)
      const cached = cacheRecommendation(item, result)
      setState({ status: "ready", item, result, analyzedAt: cached.analyzedAt })
    } catch (err) {
      setState({
        status: "error",
        item,
        message: err instanceof Error ? err.message : "Analysis failed.",
      })
    }
  }

  const reanalyzeGate = useReanalyzeGate(runAnalysis)

  return (
    <div className="flex flex-col gap-6 p-6 max-w-5xl">
      <Link href="/menu-strategy" className="flex w-fit items-center gap-1.5 text-sm text-muted-foreground hover:text-foreground">
        <ArrowLeft className="size-4" />
        Back to Menu Strategy
      </Link>

      {state.status === "loading" && (
        <Card>
          <CardContent className="h-40 animate-pulse bg-muted/50" />
        </Card>
      )}

      {state.status === "not-found" && (
        <Card>
          <CardContent className="py-10 text-center text-sm text-muted-foreground">
            This menu item isn&apos;t active, doesn&apos;t exist, or the link is invalid.
          </CardContent>
        </Card>
      )}

      {(state.status === "needs-analysis" || state.status === "error") && (
        <Card>
          <CardContent className="flex flex-col items-center gap-3 py-10 text-center">
            <div>
              <p className="font-heading text-lg font-semibold text-foreground">{state.item.name}</p>
              <p className="text-sm text-muted-foreground">
                {state.item.category} · {formatCents(state.item.price_cents)}
              </p>
            </div>
            {state.status === "error" && (
              <p className="flex items-center gap-1.5 rounded-xl bg-destructive/10 px-3 py-2 text-xs text-destructive">
                <AlertTriangle className="size-3.5 shrink-0" />
                {state.message}
              </p>
            )}
            <p className="text-sm text-muted-foreground">
              {state.status === "error" ? "Try running the analysis again." : "No analysis yet for this item."}
            </p>
            <Button onClick={() => runAnalysis(state.item)}>
              <Sparkles />
              Run Analysis
            </Button>
          </CardContent>
        </Card>
      )}

      {state.status === "analyzing" && (
        <Card>
          <CardContent className="flex flex-col items-center gap-3 py-10 text-center">
            <Loader2 className="size-6 animate-spin text-primary" />
            <p className="text-sm text-muted-foreground">
              Analyzing {state.item.name} — running Trend, Financial, and Manager agents…
            </p>
          </CardContent>
        </Card>
      )}

      {state.status === "ready" && (
        <RecommendationDetail
          item={state.item}
          result={state.result}
          analyzedAt={state.analyzedAt}
          analyzing={false}
          onReanalyze={() => reanalyzeGate.request(state.item, state.analyzedAt)}
        />
      )}

      <ReanalyzeConfirmDialog
        open={reanalyzeGate.pending !== null}
        onOpenChange={(open) => !open && reanalyzeGate.cancel()}
        daysSinceAnalysis={reanalyzeGate.pending?.days ?? 0}
        onConfirm={reanalyzeGate.confirm}
      />
    </div>
  )
}
