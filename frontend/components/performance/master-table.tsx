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
import { MessageSquare } from "lucide-react"

export function MasterTable() {
  const [selectedItem, setSelectedItem] = React.useState<any>(null)
  const [isDrawerOpen, setIsDrawerOpen] = React.useState(false)

  const handleRowClick = (item: any) => {
    setSelectedItem(item)
    setIsDrawerOpen(true)
  }

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
                {[...masterItemsData]
                  .sort((a, b) => b.totalRevenue - a.totalRevenue)
                  .map((item) => (
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
                    <TableCell className="text-right">${item.totalRevenue.toLocaleString()}</TableCell>
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
