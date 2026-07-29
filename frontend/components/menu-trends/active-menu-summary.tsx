"use client"

import { ClipboardList, Eye } from "lucide-react"

import { Button } from "@/components/ui/button"
import { Card, CardContent } from "@/components/ui/card"
import type { ActiveMenuItem } from "@/lib/api"

function categoryCounts(items: ActiveMenuItem[]) {
  const counts = new Map<string, number>()
  for (const item of items) {
    const key = item.category || "Uncategorized"
    counts.set(key, (counts.get(key) ?? 0) + 1)
  }
  return [...counts.entries()]
    .sort((a, b) => b[1] - a[1])
    .slice(0, 4)
    .map(([category, count]) => ({ category, count }))
}

export function ActiveMenuSummary({
  items,
  loading,
  error,
  onViewMenu,
}: {
  items: ActiveMenuItem[]
  loading: boolean
  error: string | null
  onViewMenu: () => void
}) {
  const categories = categoryCounts(items)

  return (
    <Card>
      <CardContent className="flex flex-wrap items-center justify-between gap-6">
        <div className="flex items-start gap-4">
          <span className="flex size-11 shrink-0 items-center justify-center rounded-2xl bg-secondary text-secondary-foreground">
            <ClipboardList className="size-5" />
          </span>
          <div>
            <p className="font-heading text-base font-semibold text-foreground">Your Active Menu is Ready</p>
            <p className="mt-1 max-w-md text-sm text-muted-foreground">
              The Menu Idea Agent will automatically scan your active menu items to find promotion
              opportunities and new item ideas.
            </p>
          </div>
        </div>

        {loading ? (
          <div className="flex gap-8 text-sm text-muted-foreground">Loading menu summary…</div>
        ) : error ? (
          <p className="text-sm text-destructive">{error}</p>
        ) : items.length === 0 ? (
          <p className="text-sm text-muted-foreground">No active menu items found.</p>
        ) : (
          <div className="flex flex-wrap gap-8">
            <Stat value={items.length} label="Active Menu Items" />
            {categories.map(({ category, count }) => (
              <Stat key={category} value={count} label={category} />
            ))}
          </div>
        )}

        <div className="flex flex-col items-start gap-1.5">
          <Button
            variant="outline"
            className="border-primary/40 text-primary hover:bg-primary/10 hover:text-primary"
            onClick={onViewMenu}
            disabled={loading || items.length === 0}
          >
            <Eye />
            View Active Menu
          </Button>
          <p className="text-xs text-muted-foreground">See all items that will be analyzed</p>
        </div>
      </CardContent>
    </Card>
  )
}

function Stat({ value, label }: { value: number; label: string }) {
  return (
    <div className="text-center">
      <p className="font-heading text-2xl font-bold text-foreground">{value}</p>
      <p className="text-xs text-muted-foreground">{label}</p>
    </div>
  )
}
