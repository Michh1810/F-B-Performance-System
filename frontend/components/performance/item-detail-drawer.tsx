"use client"

import * as React from "react"
import { Area, AreaChart, CartesianGrid, XAxis, YAxis } from "recharts"
import {
  Sheet,
  SheetContent,
  SheetDescription,
  SheetHeader,
  SheetTitle,
} from "@/components/ui/sheet"
import {
  ChartConfig,
  ChartContainer,
  ChartTooltip,
  ChartTooltipContent,
} from "@/components/ui/chart"
import { Card, CardContent } from "@/components/ui/card"
import { Badge } from "@/components/ui/badge"
import { Star, MessageSquare } from "lucide-react"
import { guestFeedbackData } from "./mock-data"

interface ItemDetailDrawerProps {
  item: any
  open: boolean
  onOpenChange: (open: boolean) => void
}

export function ItemDetailDrawer({ item, open, onOpenChange }: ItemDetailDrawerProps) {
  // Generate 12-week trend data
  const itemTrendData = React.useMemo(() => {
    if (!item) return []
    return Array.from({ length: 12 }).map((_, i) => {
      const date = new Date()
      date.setDate(date.getDate() - (11 - i) * 7) // weekly intervals
      
      const baseUnits = item.unitsSold / 4 // approx weekly
      const variance = baseUnits * 0.4 * (Math.random() * 2 - 1)
      
      return {
        week: `Wk ${12 - i}`,
        date: date.toISOString().split('T')[0],
        units: Math.max(1, Math.round(baseUnits + variance))
      }
    })
  }, [item])

  if (!item) return null

  const chartConfig = {
    units: {
      label: "Units Sold",
      color: "var(--primary)",
    },
  } satisfies ChartConfig

  const feedback = guestFeedbackData[item.id as keyof typeof guestFeedbackData] || []

  return (
    <Sheet open={open} onOpenChange={onOpenChange}>
      <SheetContent className="w-full sm:max-w-md overflow-y-auto">
        <SheetHeader className="mb-6">
          <div className="flex justify-between items-start">
            <div>
              <SheetTitle>{item.name}</SheetTitle>
              <SheetDescription className="mt-1">
                Current Menu Price: ${item.price.toFixed(2)}
              </SheetDescription>
            </div>
            <Badge variant="secondary" className="bg-muted">
              {item.category}
            </Badge>
          </div>
        </SheetHeader>

        <div className="bg-muted/50 rounded-lg p-3 text-sm mb-6 flex items-center justify-between">
          <span className="text-muted-foreground">Category Ranking</span>
          <span className="font-medium">#2 out of 18 for Revenue</span>
        </div>

        <div className="grid grid-cols-2 gap-4 mb-8">
          <Card>
            <CardContent className="p-4">
              <p className="text-sm text-muted-foreground mb-1">Total Units (30d)</p>
              <p className="text-2xl font-bold">{item.unitsSold}</p>
              <p className={`text-xs mt-1 ${item.trendUp ? 'text-green-500' : 'text-red-500'}`}>
                {item.trend} vs last period
              </p>
            </CardContent>
          </Card>
          <Card>
            <CardContent className="p-4">
              <p className="text-sm text-muted-foreground mb-1">Net Revenue</p>
              <p className="text-2xl font-bold">${item.totalRevenue.toLocaleString()}</p>
              <p className="text-xs mt-1 text-muted-foreground">
                In last 30 days
              </p>
            </CardContent>
          </Card>
        </div>

        <div className="mb-8">
          <h3 className="font-medium mb-4">Sales Trend (Last 12 Weeks)</h3>
          <ChartContainer
            config={chartConfig}
            className="aspect-auto h-[200px] w-full"
          >
            <AreaChart data={itemTrendData} margin={{ left: -20, right: 10 }}>
              <defs>
                <linearGradient id="colorUnits" x1="0" y1="0" x2="0" y2="1">
                  <stop offset="5%" stopColor="var(--color-units)" stopOpacity={0.8} />
                  <stop offset="95%" stopColor="var(--color-units)" stopOpacity={0.1} />
                </linearGradient>
              </defs>
              <CartesianGrid vertical={false} strokeDasharray="3 3" />
              <XAxis
                dataKey="week"
                tickLine={false}
                axisLine={false}
                tickMargin={8}
                tick={{ fontSize: 10 }}
              />
              <YAxis tickLine={false} axisLine={false} tickMargin={8} tick={{ fontSize: 10 }} />
              <ChartTooltip
                cursor={false}
                content={<ChartTooltipContent indicator="line" />}
              />
              <Area
                type="natural"
                dataKey="units"
                stroke="var(--color-units)"
                fill="url(#colorUnits)"
              />
            </AreaChart>
          </ChartContainer>
        </div>

        <div>
          <div className="flex items-center gap-2 mb-4">
            <MessageSquare className="h-4 w-4" />
            <h3 className="font-medium">Guest Feedback Loop</h3>
          </div>
          
          {feedback.length > 0 ? (
            <div className="space-y-4">
              {feedback.map((review, idx) => (
                <div key={idx} className="border rounded-md p-3 text-sm">
                  <div className="flex justify-between items-center mb-2">
                    <span className="font-medium text-xs text-muted-foreground">{review.source}</span>
                    <div className="flex items-center text-amber-500">
                      {Array.from({ length: 5 }).map((_, i) => (
                        <Star key={i} className={`h-3 w-3 ${i < review.rating ? "fill-current" : "text-muted opacity-30"}`} />
                      ))}
                    </div>
                  </div>
                  <p className="text-foreground leading-relaxed">"{review.text}"</p>
                  <p className="text-xs text-muted-foreground mt-2">{new Date(review.date).toLocaleDateString()}</p>
                </div>
              ))}
            </div>
          ) : (
            <div className="text-center p-8 bg-muted/30 rounded-md border border-dashed">
              <p className="text-sm text-muted-foreground">No recent guest mentions found for this item.</p>
            </div>
          )}
        </div>
      </SheetContent>
    </Sheet>
  )
}
