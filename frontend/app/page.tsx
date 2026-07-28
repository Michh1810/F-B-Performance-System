import { HeroAlert } from "@/components/overview/hero-alert"
import { WeeklyPulse } from "@/components/overview/weekly-pulse"
import { AIInsights } from "@/components/overview/ai-insights"

export default function Page() {
  return (
    <div className="flex min-h-svh flex-col p-6 max-w-7xl mx-auto w-full">
      <div className="mb-6">
        <h1 className="heading-large">Overview</h1>
        <p className="body-secondary mt-2">The Weekly Briefing</p>
      </div>

      <div className="grid grid-cols-1 lg:grid-cols-3 gap-6 mb-8 items-stretch">
        <div className="lg:col-span-1">
          <HeroAlert />
        </div>
        <div className="lg:col-span-2">
          <WeeklyPulse />

        </div>
      </div>

      <AIInsights />
    </div>
  )
}
