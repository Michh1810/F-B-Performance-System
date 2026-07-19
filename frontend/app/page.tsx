type MenuItem = {
  id: string
  name: string
  menuCategory: string
  unitsSold: number
  popularityIndex: number
  revenue: number
  foodCostPercent: number
  contributionMargin: number
  performanceCategory: string
  trendPercent: number
}

type MenuItemsResponse = {
  dateRange: {
    from: string
    to: string
  }
  items: MenuItem[]
}

async function getTopItems(): Promise<MenuItemsResponse | null> {
  const baseUrl =
    process.env.NEXT_PUBLIC_API_BASE_URL ?? "http://localhost:8080"
  const response = await fetch(`${baseUrl}/api/v1/dashboard/menu-items`, {
    cache: "no-store",
  })

  if (!response.ok) {
    return null
  }

  return (await response.json()) as MenuItemsResponse
}

export default async function Page() {
  const data = await getTopItems()
  return (
    <div className="flex min-h-svh p-6">
      <div className="flex max-w-md min-w-0 flex-col gap-4 text-sm leading-loose">
        <h1 className="font-medium">Top 5 Menu Items</h1>
        {!data ? (
          <p>Backend data unavailable.</p>
        ) : (
          <>
            <p>
              Date range: {data.dateRange.from} to {data.dateRange.to}
            </p>
            <ul className="space-y-2">
              {data.items.map((item) => (
                <li key={item.id} className="rounded border p-2">
                  <p className="font-medium">{item.name}</p>
                  <p className="text-muted-foreground">
                    Revenue: ${item.revenue.toFixed(2)} | Units: {item.unitsSold}{" "}
                    | Category: {item.performanceCategory}
                  </p>
                </li>
              ))}
            </ul>
          </>
        )}
      </div>
    </div>
  )
}
