"use client"

import { Tabs, TabsList, TabsTrigger } from "@/components/ui/tabs"
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select"

export type IdeaFilter = "all" | "promotion" | "tweak"
export type IdeaSort = "newest" | "oldest" | "views"

const SORT_LABELS: Record<IdeaSort, string> = {
  newest: "Newest First",
  oldest: "Oldest First",
  views: "Most Views",
}

export function IdeaFilters({
  counts,
  filter,
  onFilterChange,
  sort,
  onSortChange,
}: {
  counts: { all: number; promotion: number; tweak: number }
  filter: IdeaFilter
  onFilterChange: (filter: IdeaFilter) => void
  sort: IdeaSort
  onSortChange: (sort: IdeaSort) => void
}) {
  return (
    <div className="flex flex-wrap items-center justify-between gap-3">
      <Tabs value={filter} onValueChange={(value) => onFilterChange(value as IdeaFilter)}>
        <TabsList>
          <TabsTrigger value="all">All Ideas ({counts.all})</TabsTrigger>
          <TabsTrigger value="promotion">Promotions ({counts.promotion})</TabsTrigger>
          <TabsTrigger value="tweak">New Item Ideas ({counts.tweak})</TabsTrigger>
        </TabsList>
      </Tabs>

      <Select value={sort} onValueChange={(value) => onSortChange(value as IdeaSort)}>
        <SelectTrigger className="bg-card">
          <span className="text-muted-foreground">Sort by:</span>
          <SelectValue />
        </SelectTrigger>
        <SelectContent>
          {(Object.keys(SORT_LABELS) as IdeaSort[]).map((value) => (
            <SelectItem key={value} value={value}>
              {SORT_LABELS[value]}
            </SelectItem>
          ))}
        </SelectContent>
      </Select>
    </div>
  )
}
