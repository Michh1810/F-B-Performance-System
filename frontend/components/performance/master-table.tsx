"use client"

import * as React from "react"
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table"
import { Card, CardContent, CardHeader, CardTitle, CardDescription } from "@/components/ui/card"
import { Badge } from "@/components/ui/badge"
import { masterItemsData } from "./mock-data"
import { ItemDetailDrawer } from "./item-detail-drawer"
import { MessageSquare, ChevronLeft, ChevronRight } from "lucide-react"
import { Button } from "@/components/ui/button"

interface BackendMasterTableItem {
  id: string
  name: string
  category: string
  price: number
  unitsSold: number
  netRevenue: number
  guestMentions: number
  saleTrend: number
}

export function MasterTable({ masterTable }: { masterTable?: BackendMasterTableItem[] }) {
  const [selectedItem, setSelectedItem] = React.useState<any>(null)
  const [isDrawerOpen, setIsDrawerOpen] = React.useState(false)
  const [currentPage, setCurrentPage] = React.useState(1)
  
  const itemsPerPage = 15

  const handleRowClick = (item: any) => {
    setSelectedItem(item)
    setIsDrawerOpen(true)
  }

  const formatTrend = (trend: number) => {
    if (!trend) return "0%"
    const sign = trend > 0 ? "+" : ""
    return `${sign}${trend.toFixed(1)}%`
  }

  const items = (masterTable || []).map((item) => ({
    id: item.id,
    name: item.name,
    category: item.category,
    price: item.price,
    unitsSold: item.unitsSold,
    totalRevenue: item.netRevenue,
    guestMentions: item.guestMentions,
    trend: formatTrend(item.saleTrend),
    trendUp: item.saleTrend >= 0,
  }))

  const sortedItems = [...items].sort((a, b) => b.totalRevenue - a.totalRevenue)
  const totalPages = Math.max(1, Math.ceil(sortedItems.length / itemsPerPage))
  const paginatedItems = sortedItems.slice((currentPage - 1) * itemsPerPage, currentPage * itemsPerPage)

  return (
    <>
      <Card className="mb-8">
        <CardHeader>
          <CardTitle>The Master Item Table</CardTitle>
          <CardDescription>The Evidence Room: Complete performance profile merging POS and Sentiment data.</CardDescription>
        </CardHeader>
        <CardContent>
          <div className="rounded-md border">
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>Item Name & Category</TableHead>
                  <TableHead className="text-right">Menu Price</TableHead>
                  <TableHead className="text-right">Units Sold</TableHead>
                  <TableHead className="text-right">Net Revenue</TableHead>
                  <TableHead className="text-center">Guest Mentions</TableHead>
                  <TableHead className="text-right">Sale Trend</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {paginatedItems.map((item) => (
                  <TableRow
                    key={item.id}
                    className="cursor-pointer hover:bg-muted/50 transition-colors"
                    onClick={() => handleRowClick(item)}
                  >
                    <TableCell>
                      <p className="font-medium">{item.name}</p>
                      <p className="text-xs text-muted-foreground">{item.category}</p>
                    </TableCell>
                    <TableCell className="text-right">${item.price.toFixed(2)}</TableCell>
                    <TableCell className="text-right">{item.unitsSold}</TableCell>
                    <TableCell className="text-right">${item.totalRevenue.toLocaleString(undefined, {minimumFractionDigits: 2, maximumFractionDigits: 2})}</TableCell>
                    <TableCell className="text-center">
                      {item.guestMentions > 0 ? (
                        <Badge variant="secondary" className="gap-1 bg-blue-50 text-blue-700 border-blue-200 dark:bg-blue-950 dark:text-blue-300 dark:border-blue-900">
                          <MessageSquare className="h-3 w-3" />
                          {item.guestMentions}
                        </Badge>
                      ) : (
                        <span className="text-muted-foreground text-sm">-</span>
                      )}
                    </TableCell>
                    <TableCell className="text-right">
                      <Badge variant="outline" className={item.trendUp ? "text-green-600 bg-green-50 border-green-200 dark:bg-green-950 dark:border-green-800" : "text-red-600 bg-red-50 border-red-200 dark:bg-red-950 dark:border-red-800"}>
                        {item.trend}
                      </Badge>
                    </TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          </div>
          
          <div className="flex items-center justify-between mt-4">
            <p className="text-sm text-muted-foreground">
              Showing {sortedItems.length > 0 ? (currentPage - 1) * itemsPerPage + 1 : 0} to {Math.min(currentPage * itemsPerPage, sortedItems.length)} of {sortedItems.length} items
            </p>
            <div className="flex items-center space-x-2">
              <Button
                variant="outline"
                size="sm"
                onClick={() => setCurrentPage(p => Math.max(1, p - 1))}
                disabled={currentPage === 1}
              >
                <ChevronLeft className="h-4 w-4 mr-1" />
                Previous
              </Button>
              <Button
                variant="outline"
                size="sm"
                onClick={() => setCurrentPage(p => Math.min(totalPages, p + 1))}
                disabled={currentPage === totalPages}
              >
                Next
                <ChevronRight className="h-4 w-4 ml-1" />
              </Button>
            </div>
          </div>
        </CardContent>
      </Card>

      <ItemDetailDrawer
        item={selectedItem}
        open={isDrawerOpen}
        onOpenChange={setIsDrawerOpen}
      />
    </>
  )
}
