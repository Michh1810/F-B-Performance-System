"use client"

import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog"
import { Badge } from "@/components/ui/badge"
import type { ActiveMenuItem } from "@/lib/api"

export function ActiveMenuDrawer({
  open,
  onOpenChange,
  items,
}: {
  open: boolean
  onOpenChange: (open: boolean) => void
  items: ActiveMenuItem[]
}) {
  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-lg">
        <DialogHeader>
          <DialogTitle>Active Menu ({items.length})</DialogTitle>
          <DialogDescription>
            Every item here will be scanned by the Menu Idea Agent on the next run.
          </DialogDescription>
        </DialogHeader>
        <div className="-mx-6 max-h-96 overflow-y-auto px-6">
          {items.length === 0 ? (
            <p className="py-6 text-center text-sm text-muted-foreground">No active menu items.</p>
          ) : (
            <ul className="divide-y divide-border">
              {items.map((item) => (
                <li key={item.id} className="flex items-center justify-between gap-3 py-2.5">
                  <span className="text-sm font-medium text-foreground">{item.name}</span>
                  <Badge variant="outline" className="border-border text-muted-foreground">
                    {item.category}
                  </Badge>
                </li>
              ))}
            </ul>
          )}
        </div>
      </DialogContent>
    </Dialog>
  )
}
