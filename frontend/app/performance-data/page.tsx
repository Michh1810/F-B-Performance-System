import { FilterBar } from "@/components/performance/filter-bar"
import { MacroView } from "@/components/performance/macro-view"
import { MasterTable } from "@/components/performance/master-table"

export default async function PerformanceDataPage() {
  // Next.js (Node.js) sometimes tries to resolve 'localhost' to IPv6 (::1) while Docker/Go binds to IPv4 (127.0.0.1).
  // Using 127.0.0.1 explicitly prevents the "SocketError: other side closed".
  // Also, make sure to hit the correct route we defined in main.go!
  const baseUrl =
    process.env.NEXT_PUBLIC_API_BASE_URL ||
    "http://127.0.0.1:8080"

  const response = await fetch(
    `${baseUrl}/api/v1/performance-dashboard`,
    { cache: "no-store" }
  )
  
  const dashboardData = await response.json()
  return (
    <div className="flex min-h-svh flex-col p-6 max-w-7xl mx-auto w-full">
      <div className="mb-8">
        <h1 className="heading-large">Performance Data</h1>
      </div>
      <FilterBar />
      <MacroView kpis={dashboardData.kpis} revenueClasses={dashboardData.revenueClasses} />
      <MasterTable masterTable={dashboardData.masterTable} />
    </div>
  )
}
