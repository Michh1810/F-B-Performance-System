"use client"

import { Lightbulb } from "lucide-react"

import { Card, CardContent } from "@/components/ui/card"
import type { MenuIdea } from "@/lib/api"
import { IdeaCard } from "./idea-card"

export function IdeaList({
  ideas,
  loading,
  error,
  onReview,
  onDismiss,
  dismissingId,
}: {
  ideas: MenuIdea[]
  loading: boolean
  error: string | null
  onReview: (idea: MenuIdea) => void
  onDismiss: (idea: MenuIdea) => void
  dismissingId: string | null
}) {
  if (loading) {
    return (
      <div className="flex flex-col gap-4">
        {[0, 1, 2].map((i) => (
          <Card key={i}>
            <CardContent className="h-24 animate-pulse bg-muted/50" />
          </Card>
        ))}
      </div>
    )
  }

  if (error) {
    return (
      <Card>
        <CardContent className="py-8 text-center text-sm text-destructive">{error}</CardContent>
      </Card>
    )
  }

  if (ideas.length === 0) {
    return (
      <Card>
        <CardContent className="flex flex-col items-center gap-2 py-10 text-center">
          <Lightbulb className="size-6 text-muted-foreground" />
          <p className="text-sm font-medium text-foreground">No ideas match this filter</p>
          <p className="text-sm text-muted-foreground">
            Run the Menu Idea Agent or adjust your filters to see results here.
          </p>
        </CardContent>
      </Card>
    )
  }

  return (
    <div className="flex flex-col gap-4">
      {ideas.map((idea) => (
        <IdeaCard
          key={idea.id}
          idea={idea}
          onReview={() => onReview(idea)}
          onDismiss={() => onDismiss(idea)}
          dismissing={dismissingId === idea.id}
        />
      ))}
    </div>
  )
}
