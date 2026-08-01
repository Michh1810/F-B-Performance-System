"use client"

import Link from "next/link"
import { AlertTriangle, Check, Info, Loader2, RefreshCw, Sparkles } from "lucide-react"

import { Button } from "@/components/ui/button"
import { Card, CardContent } from "@/components/ui/card"
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table"
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip"
import type { ActiveMenuItem } from "@/lib/api"
import { DecisionBadge } from "./decision-badge"
import { formatCents, formatRelativeTime } from "./format"
import type { RowState } from "./types"

const columnHeadClassName = "text-xs font-semibold tracking-wide text-primary uppercase"

export function MenuStrategyTable({
  items,
  loading,
  error,
  rowStates,
  onAnalyze,
  onReanalyzeRequest,
}: {
  items: ActiveMenuItem[]
  loading: boolean
  error: string | null
  rowStates: Record<string, RowState>
  onAnalyze: (item: ActiveMenuItem) => void
  onReanalyzeRequest: (item: ActiveMenuItem, analyzedAt: string) => void
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

  if (items.length === 0) {
    return (
      <Card>
        <CardContent className="py-10 text-center text-sm text-muted-foreground">
          No active menu items to analyze.
        </CardContent>
      </Card>
    )
  }

  return (
    <Card size="sm" className="overflow-hidden">
      <Table>
        <TableHeader>
          <TableRow>
            <TableHead className={columnHeadClassName}>Item</TableHead>
            <TableHead className={columnHeadClassName}>Category</TableHead>
            <TableHead className={columnHeadClassName}>Price</TableHead>
            <TableHead className={columnHeadClassName}>Decision</TableHead>
            <TableHead className={`${columnHeadClassName} text-center`}>Action</TableHead>
          </TableRow>
        </TableHeader>
        <TableBody>
          {items.map((item) => {
            const row = rowStates[item.id] ?? { status: "idle" as const }
            return (
              <TableRow key={item.id}>
                <TableCell className="font-medium text-foreground">{item.name}</TableCell>
                <TableCell className="text-muted-foreground">{item.category}</TableCell>
                <TableCell className="text-muted-foreground">{formatCents(item.price_cents)}</TableCell>
                <TableCell>
                  {row.status === "loading" && (
                    <span className="flex items-center gap-1.5 text-xs text-muted-foreground">
                      <Loader2 className="size-3.5 animate-spin" />
                      Analyzing…
                    </span>
                  )}
                  {row.status === "error" && (
                    <span className="flex items-center gap-1.5 text-xs text-destructive" title={row.error}>
                      <AlertTriangle className="size-3.5" />
                      Failed
                    </span>
                  )}
                  {row.status === "done" && <DecisionBadge decision={row.result.decision} />}
                  {row.status === "idle" && (
                    <span className="text-xs text-muted-foreground">
                      {item.has_sufficient_history ? "Not analyzed" : "New item"}
                    </span>
                  )}
                </TableCell>
                <TableCell>
                  {row.status === "done" ? (
                    <div className="flex flex-col items-center gap-1.5">
                      <span className="flex items-center gap-1 text-xs text-emerald-700" title={row.analyzedAt}>
                        <Check className="size-3.5" />
                        Updated {formatRelativeTime(row.analyzedAt)}
                      </span>
                      <div className="flex items-center justify-center gap-2">
                        <Button size="sm" asChild>
                          <Link href={`/menu-strategy/${item.id}`}>View</Link>
                        </Button>
                        <Button
                          size="sm"
                          variant="outline"
                          className="border-border"
                          onClick={() => onReanalyzeRequest(item, row.analyzedAt)}
                        >
                          <RefreshCw />
                          Reanalyze
                        </Button>
                      </div>
                    </div>
                  ) : row.status !== "loading" && !item.has_sufficient_history ? (
                    <div className="flex justify-center">
                      <Tooltip>
                        <TooltipTrigger asChild>
                          <span className="flex items-center gap-1.5 text-xs text-muted-foreground">
                            <Info className="size-3.5 shrink-0" />
                            Not enough data yet
                          </span>
                        </TooltipTrigger>
                        <TooltipContent>
                          This item is new and doesn&apos;t have enough sales history to analyze — it needs at
                          least 30 days of transactions first.
                        </TooltipContent>
                      </Tooltip>
                    </div>
                  ) : (
                    <div className="flex justify-center">
                      <Button size="sm" onClick={() => onAnalyze(item)} disabled={row.status === "loading"}>
                        {row.status === "loading" ? <Loader2 className="animate-spin" /> : <Sparkles />}
                        {row.status === "error" ? "Retry" : "Analyze"}
                      </Button>
                    </div>
                  )}
                </TableCell>
              </TableRow>
            )
          })}
        </TableBody>
      </Table>
    </Card>
  )
}
