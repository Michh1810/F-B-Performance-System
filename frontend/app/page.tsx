import { HeroAlert } from "@/components/overview/hero-alert"
import { WeeklyPulse } from "@/components/overview/weekly-pulse"
import { AIInsights } from "@/components/overview/ai-insights"

export default async function Page() {
  const baseUrl = process.env.NEXT_PUBLIC_API_BASE_URL!
  const response = await fetch(
    `${baseUrl}/api/v1/overview`,
    { cache: "no-store" }
  )
  let overviewData = null
  try {
    overviewData = await response.json()
  } catch (err) {
    console.error("Failed to parse overview JSON", err)
  }

  return (
    <div className="flex min-h-svh flex-col p-6 max-w-7xl mx-auto w-full">
      <div className="mb-6">
        <h1 className="heading-large">Overview</h1>
        <p className="body-secondary mt-2">The Weekly Briefing</p>
      </div>

      {overviewData ? (
        <>
          <div className="grid grid-cols-1 lg:grid-cols-3 gap-6 mb-8 items-stretch">
            <div className="lg:col-span-1">
              <HeroAlert criticalAlert={overviewData.criticalAlert} />
            </div>
            <div className="lg:col-span-2">
              <WeeklyPulse pulse={overviewData.weeklyPulse} />
            </div>
          </div>

          <AIInsights insights={overviewData.aiInsights} />
        </>
      ) : (
        <div className="flex items-center justify-center h-64">
          <p className="text-muted-foreground">Loading overview data...</p>
        </div>
      )}
    </div>
  )
}
