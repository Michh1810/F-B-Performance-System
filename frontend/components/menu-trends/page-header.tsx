"use client"

import { HelpCircle } from "lucide-react"

import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
} from "@/components/ui/dialog"

const STEPS = [
  {
    title: "1. Describe your restaurant",
    body: "Your description grounds the AI's hashtag suggestions in your cuisine, vibe, and signature dishes.",
  },
  {
    title: "2. Discover trending hashtags",
    body: "Those hashtags are used to pull trending TikTok videos, which get turned into searchable trend signals.",
  },
  {
    title: "3. Scan your active menu",
    body: "The agent embeds every active menu item and searches for closely related trend signals (distance < 0.35).",
  },
  {
    title: "4. Generate ideas",
    body: "Signals naming an existing item become promotion ideas (no LLM needed). Adjacent signals are sent to the LLM, which proposes up to 2 new item ideas per menu item.",
  },
  {
    title: "5. Review and approve",
    body: 'Every idea starts as "new". You approve promotions as-is, or price a new item yourself before it becomes a real menu item — the AI never sets pricing.',
  },
]

export function PageHeader() {
  return (
    <header className="flex flex-wrap items-start justify-between gap-4 px-8 pt-8">
      <div>
        <div className="flex items-center gap-2.5">
          <h1 className="font-heading text-3xl font-bold text-foreground">Menu Trends &amp; Ideas</h1>
          <Badge className="bg-secondary text-secondary-foreground">Beta</Badge>
        </div>
        <p className="mt-1.5 max-w-xl text-sm text-muted-foreground">
          Discover what&apos;s trending on TikTok and get AI-powered ideas to promote or expand your menu.
        </p>
      </div>

      <Dialog>
        <DialogTrigger asChild>
          <Button variant="outline" className="border-border bg-card">
            <HelpCircle />
            How it works
          </Button>
        </DialogTrigger>
        <DialogContent className="sm:max-w-lg">
          <DialogHeader>
            <DialogTitle>How the Menu Idea Agent works</DialogTitle>
            <DialogDescription>
              From a restaurant description to reviewable menu ideas, end to end.
            </DialogDescription>
          </DialogHeader>
          <ol className="space-y-4">
            {STEPS.map((step) => (
              <li key={step.title}>
                <p className="text-sm font-semibold text-foreground">{step.title}</p>
                <p className="mt-0.5 text-sm text-muted-foreground">{step.body}</p>
              </li>
            ))}
          </ol>
        </DialogContent>
      </Dialog>
    </header>
  )
}
