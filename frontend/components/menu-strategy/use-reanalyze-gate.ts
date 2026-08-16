"use client"

import { useState } from "react"

import type { ActiveMenuItem } from "@/lib/api"
import { REANALYZE_FRESHNESS_DAYS, daysSince } from "@/lib/recommendation-cache"

// Shared by the table (menu-action-plan.tsx) and the detail page — both let
// the user re-analyze an already-analyzed item, and both should gate a
// too-soon re-run behind the same confirmation prompt.
export function useReanalyzeGate(onRun: (item: ActiveMenuItem) => void) {
  const [pending, setPending] = useState<{ item: ActiveMenuItem; days: number } | null>(null)

  function request(item: ActiveMenuItem, analyzedAt?: string) {
    if (analyzedAt) {
      const days = daysSince(analyzedAt)
      if (days < REANALYZE_FRESHNESS_DAYS) {
        setPending({ item, days })
        return
      }
    }
    onRun(item)
  }

  function confirm() {
    if (!pending) return
    onRun(pending.item)
    setPending(null)
  }

  function cancel() {
    setPending(null)
  }

  return { pending, request, confirm, cancel }
}
