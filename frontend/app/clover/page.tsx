type CloverLineItem = {
  id: string
  name: string
  price: number
  item: {
    id: string
  }
}

type CloverOrder = {
  id: string
  total: number
  paymentState: string
  state: string
  createdTime: number
  lineItems: {
    elements: CloverLineItem[]
  }
}

type CloverOrdersResponse = {
  elements: CloverOrder[]
}

async function getCloverOrders(): Promise<CloverOrdersResponse | null> {
  const baseUrl = process.env.NEXT_PUBLIC_API_BASE_URL ?? "http://localhost:8080"
  
  // Fetch from the new Go API route!
  const response = await fetch(`${baseUrl}/api/clover`, {
    cache: "no-store",
  })

  if (!response.ok) {
    return null
  }

  return (await response.json()) as CloverOrdersResponse
}

export default async function CloverPage() {
  const data = await getCloverOrders()

  return (
    <div className="flex min-h-svh p-6 bg-slate-50">
      <div className="flex max-w-2xl min-w-0 flex-col gap-6 text-sm leading-loose w-full">
        <h1 className="text-2xl font-bold tracking-tight">Today's Clover Sales</h1>
        
        {!data ? (
          <div className="p-4 rounded-lg bg-red-50 text-red-600 border border-red-200">
            Backend data unavailable. Ensure the Go server is running!
          </div>
        ) : (
          <div className="space-y-4">
            <p className="font-medium text-slate-600">
              Found {data.elements?.length || 0} orders today.
            </p>

            <ul className="grid gap-4 sm:grid-cols-2">
              {data.elements?.map((order) => (
                <li key={order.id} className="rounded-xl border bg-white shadow-sm p-4 hover:shadow-md transition-shadow">
                  <div className="flex justify-between items-center mb-2">
                    <span className="font-bold">Order: {order.id.slice(-5)}</span>
                    <span className="bg-green-100 text-green-800 text-xs font-semibold px-2 py-1 rounded">
                      ${(order.total / 100).toFixed(2)}
                    </span>
                  </div>
                  
                  <div className="text-slate-500 text-xs mb-3">
                    State: {order.state} | Payment: {order.paymentState}
                  </div>

                  <div className="border-t pt-2 mt-2">
                    <p className="text-xs font-semibold text-slate-400 mb-1">LINE ITEMS</p>
                    <ul className="space-y-1">
                      {order.lineItems?.elements?.map((item) => (
                        <li key={item.id} className="flex justify-between text-xs">
                          <span className="truncate pr-2">{item.name}</span>
                          <span className="font-medium">${(item.price / 100).toFixed(2)}</span>
                        </li>
                      ))}
                      {(!order.lineItems?.elements || order.lineItems.elements.length === 0) && (
                        <li className="text-xs italic text-slate-400">No items on ticket</li>
                      )}
                    </ul>
                  </div>
                </li>
              ))}
            </ul>
          </div>
        )}
      </div>
    </div>
  )
}
