import { DashboardShell } from "@/components/layout/dashboard-shell"
import { MenuTrendsIdeasPage } from "@/components/menu-trends/menu-trends-ideas-page"

export default function AIRecommendationsPage() {
  return (
    <DashboardShell active="AI Recommendations">
      <MenuTrendsIdeasPage />
    </DashboardShell>
  )
}
