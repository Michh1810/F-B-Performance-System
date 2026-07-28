"use client"

import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card"
import { ArrowDownIcon, ArrowUpIcon, DollarSign, ShoppingBag, CreditCard, PieChart } from "lucide-react"
import { Bar, BarChart, XAxis, YAxis, CartesianGrid, Tooltip as RechartsTooltip, ResponsiveContainer } from "recharts"
import { ChartConfig, ChartContainer, ChartTooltipContent } from "@/components/ui/chart"
import { macroData, revenueClassData } from "./mock-data"

const chartConfig = {
  revenue: {
    label: "Revenue",
  },
  dineIn: {
    label: "Dine-in",
    color: "hsl(var(--chart-1))",
  },
  takeout: {
    label: "Takeout",
    color: "hsl(var(--chart-2))",
  },
  thirdParty: {
    label: "Third-Party",
    color: "hsl(var(--chart-3))",
  },
} satisfies ChartConfig

export function MacroView() {
  const { netSales, orders, aov, categoryDominance } = macroData

  return (
    <div className="mb-8 space-y-6">
      <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-4">
        <Card>
          <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
            <CardTitle className="text-sm font-medium">{netSales.label}</CardTitle>
            <DollarSign className="h-4 w-4 text-muted-foreground" />
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold">${netSales.value.toLocaleString()}</div>
            <p className="text-xs text-muted-foreground mt-1 flex items-center gap-1">
              <span className={netSales.trendUp ? "text-green-500" : "text-red-500"}>
                {netSales.trendUp ? <ArrowUpIcon className="h-3 w-3 inline" /> : <ArrowDownIcon className="h-3 w-3 inline" />}
                {netSales.trend}
              </span>{" "}
              vs last week
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
              vs last week
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
              vs last week
            </p>
          </CardContent>
        </Card>

        <Card>
          <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
            <CardTitle className="text-sm font-medium">{categoryDominance.label}</CardTitle>
            <PieChart className="h-4 w-4 text-muted-foreground" />
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold">{categoryDominance.category}</div>
            <p className="text-xs text-muted-foreground mt-1">
              {categoryDominance.description}
            </p>
          </CardContent>
        </Card>
      </div>

      <Card>
        <CardHeader>
          <CardTitle>Revenue Class Breakdown</CardTitle>
        </CardHeader>
        <CardContent>
          <ChartContainer config={chartConfig} className="h-[200px] w-full">
            <ResponsiveContainer width="100%" height="100%">
              <BarChart data={revenueClassData} margin={{ top: 20, right: 20, left: 20, bottom: 0 }}>
                <CartesianGrid strokeDasharray="3 3" vertical={false} />
                <XAxis dataKey="class" axisLine={false} tickLine={false} tickMargin={10} />
                <YAxis 
                  type="number" 
                  axisLine={false} 
                  tickLine={false} 
                  tickFormatter={(value) => `$${value / 1000}k`} 
                />
                <RechartsTooltip cursor={{fill: 'transparent'}} content={<ChartTooltipContent hideLabel />} />
                <Bar dataKey="revenue" radius={[4, 4, 0, 0]} barSize={40} />
              </BarChart>
            </ResponsiveContainer>
          </ChartContainer>
        </CardContent>
      </Card>
    </div>
  )
}
