import { AlertTriangle, AlertCircle } from "lucide-react"
import { Card, CardContent } from "@/components/ui/card"
import { Button } from "@/components/ui/button"

interface HeroAlertProps {
  criticalAlert: {
    active: boolean
    type: string
    title: string
    message: string
  } | null
}

export function HeroAlert({ criticalAlert }: HeroAlertProps) {
  if (!criticalAlert || !criticalAlert.active) return null

  const isDestructive = criticalAlert.type === "destructive"

  return (
    <Card className={`h-full ${isDestructive ? 'border-red-200 bg-red-50 dark:bg-red-950/20 dark:border-red-900/50' : 'border-amber-200 bg-amber-50 dark:bg-amber-950/20 dark:border-amber-900/50'}`}>
      <CardContent className="pt-6 flex flex-col h-full justify-between gap-4">
        <div>
          <div className="flex items-center gap-2 mb-3">
            {isDestructive ? <AlertCircle className="h-6 w-6 text-red-600" /> : <AlertTriangle className="h-6 w-6 text-amber-600 dark:text-amber-400" />}
            <h3 className="font-bold text-xl text-foreground">{criticalAlert.title}</h3>
          </div>
          <p className="text-muted-foreground text-sm leading-relaxed">
            {criticalAlert.message}
          </p>
        </div>
        <Button variant={isDestructive ? "destructive" : "default"} className={!isDestructive ? "bg-amber-600 hover:bg-amber-700 text-white" : ""}>
          Review Now
        </Button>
      </CardContent>
    </Card>
  )
}
