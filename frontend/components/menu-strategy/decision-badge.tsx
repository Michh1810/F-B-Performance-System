import { Badge } from "@/components/ui/badge"
import { cn } from "@/lib/utils"
import type { Decision } from "@/lib/api"

// LAUNCH/CUT/REPRICE are the Manager Agent's only three possible calls on
// an existing item — see Decision's doc comment in lib/api/types.ts.
// Labeled for a human reviewing an *existing* menu item, not a new one.
const DECISION_LABELS: Record<Decision, string> = {
  LAUNCH: "Keep",
  CUT: "Drop",
  REPRICE: "Reprice",
}

const DECISION_STYLES: Record<Decision, string> = {
  LAUNCH: "bg-emerald-100 text-emerald-800",
  CUT: "bg-destructive/10 text-destructive",
  REPRICE: "bg-secondary text-secondary-foreground",
}

export function DecisionBadge({ decision, className }: { decision: Decision; className?: string }) {
  return <Badge className={cn(DECISION_STYLES[decision], className)}>{DECISION_LABELS[decision]}</Badge>
}
