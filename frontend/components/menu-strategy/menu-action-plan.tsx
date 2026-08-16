"use client"

import { useEffect, useState } from "react"
import { useRouter } from "next/navigation"

import { getRecommendation, listActiveMenuItems, type ActiveMenuItem } from "@/lib/api"
import { cacheRecommendation, getCachedRecommendation, listCachedRecommendationIds } from "@/lib/recommendation-cache"
import { MenuStrategyTable } from "./menu-strategy-table"
import { ReanalyzeConfirmDialog } from "./reanalyze-confirm-dialog"
import { useReanalyzeGate } from "./use-reanalyze-gate"
import type { RowState } from "./types"

export function MenuActionPlan() {
  const router = useRouter()

  const [items, setItems] = useState<ActiveMenuItem[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [rowStates, setRowStates] = useState<Record<string, RowState>>({})

  useEffect(() => {
    const controller = new AbortController()
    listActiveMenuItems(controller.signal)
      .then((loadedItems) => {
        setItems(loadedItems)
        // Restore "Done" status for anything already analyzed this session
        // (e.g. the user navigated to a detail page and came back).
        const cachedIds = new Set(listCachedRecommendationIds())
        const restored: Record<string, RowState> = {}
        for (const item of loadedItems) {
          if (!cachedIds.has(item.id)) continue
          const cached = getCachedRecommendation(item.id)
          if (cached) restored[item.id] = { status: "done", result: cached.result, analyzedAt: cached.analyzedAt }
        }
        if (Object.keys(restored).length > 0) {
          setRowStates((prev) => ({ ...restored, ...prev }))
        }
      })
      .catch((err: unknown) => {
        if (err instanceof DOMException && err.name === "AbortError") return
        setError(err instanceof Error ? err.message : "Failed to load menu items.")
      })
      .finally(() => setLoading(false))
    return () => controller.abort()
  }, [])

  async function handleAnalyze(item: ActiveMenuItem) {
    setRowStates((prev) => ({ ...prev, [item.id]: { status: "loading" } }))
    try {
      const result = await getRecommendation(item.id, item.name)
      const cached = cacheRecommendation(item, result)
      setRowStates((prev) => ({
        ...prev,
        [item.id]: { status: "done", result, analyzedAt: cached.analyzedAt },
      }))
      router.push(`/menu-strategy/${item.id}`)
    } catch (err) {
      setRowStates((prev) => ({
        ...prev,
        [item.id]: { status: "error", error: err instanceof Error ? err.message : "Analysis failed." },
      }))
    }
  }

  const reanalyzeGate = useReanalyzeGate(handleAnalyze)

  return (
    <>
      <MenuStrategyTable
        items={items}
        loading={loading}
        error={error}
        rowStates={rowStates}
        onAnalyze={handleAnalyze}
        onReanalyzeRequest={(item, analyzedAt) => reanalyzeGate.request(item, analyzedAt)}
      />
      <ReanalyzeConfirmDialog
        open={reanalyzeGate.pending !== null}
        onOpenChange={(open) => !open && reanalyzeGate.cancel()}
        daysSinceAnalysis={reanalyzeGate.pending?.days ?? 0}
        onConfirm={reanalyzeGate.confirm}
      />
    </>
  )
}
