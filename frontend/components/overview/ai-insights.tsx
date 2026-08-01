import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card"
import { Badge } from "@/components/ui/badge"
import { TrendingUp, DollarSign, AlertCircle, MessageSquare } from "lucide-react"

interface AIInsight {
  id: string
  pillar: string
  title: string
  icon: string
  suggestion: string
}

export function AIInsights({ insights }: { insights: AIInsight[] }) {
  if (!insights || insights.length === 0) return null
  
  const getIcon = (iconName: string) => {
    switch (iconName) {
      case "trendingUp": return <TrendingUp className="h-4 w-4 text-primary" />
      case "dollarSign": return <DollarSign className="h-4 w-4 text-green-500" />
      case "alertCircle": return <AlertCircle className="h-4 w-4 text-red-500" />
      case "messageSquare": return <MessageSquare className="h-4 w-4 text-blue-500" />
      default: return null
    }
  }

  return (
    <Card className="h-full flex flex-col">
      <CardHeader className="pb-4">
        <div className="flex items-center justify-between">
          <CardTitle className="text-lg">AI Insights Checklist</CardTitle>
          <Badge variant="outline" className="bg-primary/5 text-primary border-primary/20">
            Updated Weekly
          </Badge>
        </div>
      </CardHeader>
      <CardContent className="flex-1 pb-6">
        <div className="flex flex-col sm:flex-row gap-4 sm:gap-0 h-full divide-y sm:divide-y-0 sm:divide-x">
          {insights.map((insight) => (
            <div key={insight.id} className="flex-1 flex flex-col sm:px-4 first:sm:pl-0 last:sm:pr-0 py-4 sm:py-0">
              <div className="flex items-center gap-2 mb-3">
                <div className="p-1.5 bg-muted rounded-md shrink-0">
                  {getIcon(insight.icon)}
                </div>
                <p className="text-xs font-semibold text-muted-foreground truncate" title={insight.pillar}>{insight.pillar}</p>
              </div>
              <h4 className="text-sm font-bold mb-2 leading-tight">{insight.title}</h4>
              <p className="text-xs text-muted-foreground leading-relaxed">
                {insight.suggestion.split(":").map((part, index) => {
                  if (index === 0) {
                    return <strong key={index} className="text-foreground">{part}:</strong>
                  }
                  return <span key={index}>{part}</span>
                })}
              </p>
            </div>
          ))}
        </div>
      </CardContent>
    </Card>
  )
}
