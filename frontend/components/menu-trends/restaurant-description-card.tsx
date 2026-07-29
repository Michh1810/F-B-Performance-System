"use client"

import { Info, Loader2 } from "lucide-react"

import { Button } from "@/components/ui/button"
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card"

const MAX_LENGTH = 1000
const EXAMPLE_PLACEHOLDER =
  "We are a modern Mexican street food restaurant in downtown Austin, Texas. We serve authentic tacos, burritos, quesadillas, loaded fries, and refreshing drinks. Our vibe is casual, vibrant, and perfect for late night eats."

export function RestaurantDescriptionCard({
  value,
  onChange,
  onSave,
  dirty,
  saving,
  loading,
  disabled = false,
}: {
  value: string
  onChange: (value: string) => void
  onSave: () => void
  dirty: boolean
  saving: boolean
  loading: boolean
  disabled?: boolean
}) {
  return (
    <Card className="flex-1">
      <CardHeader>
        <div className="flex items-start justify-between gap-3">
          <div>
            <CardTitle className="text-base">1. Restaurant Description</CardTitle>
            <p className="mt-1 text-sm text-muted-foreground">
              Describe your restaurant to generate relevant hashtags.
            </p>
          </div>
          {dirty && (
            <Button size="sm" onClick={onSave} disabled={saving || disabled}>
              {saving ? <Loader2 className="animate-spin" /> : null}
              Save
            </Button>
          )}
        </div>
      </CardHeader>
      <CardContent className="flex flex-col gap-3">
        <textarea
          value={value}
          onChange={(e) => onChange(e.target.value.slice(0, MAX_LENGTH))}
          placeholder={EXAMPLE_PLACEHOLDER}
          disabled={loading || disabled}
          rows={7}
          className="w-full resize-none rounded-2xl border border-border bg-background px-3.5 py-3 text-sm leading-relaxed text-foreground placeholder:text-muted-foreground/70 focus:border-ring focus:outline-none disabled:opacity-60"
        />
        <p className="self-end text-xs text-muted-foreground">
          {value.length} / {MAX_LENGTH}
        </p>

        <div className="flex items-start gap-2.5 rounded-2xl bg-secondary/50 px-4 py-3">
          <Info className="mt-0.5 size-4 shrink-0 text-secondary-foreground/70" />
          <p className="text-xs leading-relaxed text-secondary-foreground/80">
            <span className="font-semibold">Tip:</span> Include your cuisine, signature dishes, location,
            atmosphere, and unique selling points for better hashtag generation.
          </p>
        </div>
      </CardContent>
    </Card>
  )
}
