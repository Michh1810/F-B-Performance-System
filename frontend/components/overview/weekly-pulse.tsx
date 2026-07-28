import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card"
import { Badge } from "@/components/ui/badge"
import { ArrowDownIcon, ArrowUpIcon, DollarSign, ShoppingBag, CreditCard, Star } from "lucide-react"
import { format, subDays } from "date-fns"
import { weeklyPulseData } from "./mock-data"

export function WeeklyPulse() {
  const { revenue, orders, aov, sentiment } = weeklyPulseData

  const today = new Date()
  const lastWeek = subDays(today, 7)
  const dateRangeStr = `${format(lastWeek, "MMM d")} - ${format(today, "MMM d, yyyy")}`

  return (
    <div className="mb-8">
      <div className="flex items-center justify-between mb-4">
        <h2 className="heading-small">Weekly Health Pulse</h2>
        <Badge variant="outline" className="text-muted-foreground font-normal">
          Last 7 Days ({dateRangeStr})
        </Badge>
      </div>
      <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-4">
        <Card>
          <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
            <CardTitle className="text-sm font-medium">{revenue.label}</CardTitle>
            <DollarSign className="h-4 w-4 text-muted-foreground" />
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold">${revenue.value.toLocaleString()}</div>
            <p className="text-xs text-muted-foreground mt-1 flex items-center gap-1">
              <span className={revenue.trendUp ? "text-green-500" : "text-red-500"}>
                {revenue.trendUp ? <ArrowUpIcon className="h-3 w-3 inline" /> : <ArrowDownIcon className="h-3 w-3 inline" />}
                {revenue.trend}
              </span>{" "}
              vs. {revenue.vs}
            </p>
          </CardContent>
        </Card>

        <Card>
          <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
            <CardTitle className="text-sm font-medium">{aov.label}</CardTitle>
            <CreditCard className="h-4 w-4 text-muted-foreground" />
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold">${aov.value.toLocaleString()}</div>
            <p className="text-xs text-muted-foreground mt-1 flex items-center gap-1">
              <span className={aov.trendUp ? "text-green-500" : "text-red-500"}>
                {aov.trendUp ? <ArrowUpIcon className="h-3 w-3 inline" /> : <ArrowDownIcon className="h-3 w-3 inline" />}
                {aov.trend}
              </span>{" "}
              vs. {aov.vs}
            </p>
          </CardContent>
        </Card>

        <Card>
          <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
            <CardTitle className="text-sm font-medium">{orders.label}</CardTitle>
            <ShoppingBag className="h-4 w-4 text-muted-foreground" />
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold">{orders.value.toLocaleString()}</div>
            <p className="text-xs text-muted-foreground mt-1 flex items-center gap-1">
              <span className={orders.trendUp ? "text-green-500" : "text-red-500"}>
                {orders.trendUp ? <ArrowUpIcon className="h-3 w-3 inline" /> : <ArrowDownIcon className="h-3 w-3 inline" />}
                {orders.trend}
              </span>{" "}
              vs. {orders.vs}
            </p>
          </CardContent>
        </Card>

        <Card>
          <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
            <CardTitle className="text-sm font-medium">{sentiment.label}</CardTitle>
            <Star className="h-4 w-4 text-muted-foreground" />
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold">{sentiment.value.toFixed(1)} <span className="text-sm text-muted-foreground font-normal">/ 5.0</span></div>
            <p className="text-xs text-muted-foreground mt-1 flex items-center gap-1">
              <span className={sentiment.trendUp ? "text-green-500" : "text-red-500"}>
                {sentiment.trendUp ? <ArrowUpIcon className="h-3 w-3 inline" /> : <ArrowDownIcon className="h-3 w-3 inline" />}
                {sentiment.trend}
              </span>{" "}
              vs. {sentiment.vs}
            </p>
          </CardContent>
        </Card>
      </div>
    </div>
  )
}
