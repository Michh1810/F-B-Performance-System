"use client"

import { Button } from "@/components/ui/button"
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog"

export function ReanalyzeConfirmDialog({
  open,
  onOpenChange,
  daysSinceAnalysis,
  onConfirm,
}: {
  open: boolean
  onOpenChange: (open: boolean) => void
  daysSinceAnalysis: number
  onConfirm: () => void
}) {
  const ageLabel = daysSinceAnalysis <= 0 ? "today" : `${daysSinceAnalysis} day${daysSinceAnalysis === 1 ? "" : "s"} ago`

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-sm">
        <DialogHeader>
          <DialogTitle>Re-analyze so soon?</DialogTitle>
          <DialogDescription>
            This item was last analyzed {ageLabel}. Trend and financial signals rarely shift much in that
            window, so the result is likely to come back nearly the same — re-running it spends another
            live LLM call for little new information.
          </DialogDescription>
        </DialogHeader>
        <DialogFooter>
          <Button variant="outline" className="border-border" onClick={() => onOpenChange(false)}>
            Cancel
          </Button>
          <Button
            onClick={() => {
              onOpenChange(false)
              onConfirm()
            }}
          >
            Re-analyze Anyway
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}
