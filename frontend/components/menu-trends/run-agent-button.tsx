"use client"

import { AlertTriangle, Loader2, Play } from "lucide-react"

import { Button } from "@/components/ui/button"

export type RunPhase = "idle" | "running" | "error"

export function RunAgentButton({
  phase,
  loadingText,
  errorMessage,
  hasHashtags,
  onRun,
}: {
  phase: RunPhase
  loadingText: string
  errorMessage: string | null
  hasHashtags: boolean
  onRun: () => void
}) {
  const running = phase === "running"

  return (
    <div className="flex flex-col items-center gap-3 px-8 py-2 text-center">
      <Button
        size="lg"
        className="h-12 rounded-2xl bg-primary px-8 text-base font-semibold text-primary-foreground hover:bg-primary/90"
        onClick={onRun}
        disabled={running || !hasHashtags}
        title={!hasHashtags ? "Add at least one hashtag before running the agent" : undefined}
      >
        {running ? <Loader2 className="animate-spin" /> : <Play />}
        {running ? loadingText : "Run Menu Idea Agent"}
      </Button>

      <p className="text-sm text-muted-foreground">
        This will fetch TikTok trends and generate menu ideas.
      </p>
      <p className="rounded-full bg-secondary px-3 py-1 text-xs font-medium text-secondary-foreground">
        Estimated time: 1–2 minutes
      </p>

      {!hasHashtags && phase !== "error" && (
        <p className="mt-1 flex items-center gap-1.5 rounded-xl bg-secondary/60 px-3 py-2 text-xs text-secondary-foreground">
          <AlertTriangle className="size-3.5 shrink-0" />
          Add at least one hashtag above before running the agent.
        </p>
      )}

      {phase === "error" && errorMessage && (
        <p className="mt-1 flex items-center gap-1.5 rounded-xl bg-destructive/10 px-3 py-2 text-xs text-destructive">
          <AlertTriangle className="size-3.5 shrink-0" />
          {errorMessage}
        </p>
      )}
    </div>
  )
}
