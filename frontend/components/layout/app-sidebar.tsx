"use client"

import Link from "next/link"
import {
  BarChart3,
  ClipboardCheck,
  FileText,
  LayoutDashboard,
  Package,
  Settings,
  Sparkles,
  UtensilsCrossed,
} from "lucide-react"

import { cn } from "@/lib/utils"

type NavItem = {
  label: string
  icon: React.ComponentType<{ className?: string }>
  href?: string
}

const NAV_ITEMS: NavItem[] = [
  { label: "Menu", icon: UtensilsCrossed },
  { label: "Sales Insights", icon: BarChart3 },
  { label: "AI Recommendations", icon: Sparkles, href: "/ai-recommendations" },
  { label: "Ideas Review", icon: ClipboardCheck },
  { label: "Reports", icon: FileText },
  { label: "Settings", icon: Settings },
]

export function AppSidebar({ active }: { active: string }) {
  return (
    <aside className="sticky top-0 flex h-svh w-64 shrink-0 flex-col overflow-y-auto bg-sidebar text-sidebar-foreground">
      <div className="flex items-center gap-2.5 px-6 py-6">
        <span className="flex size-9 shrink-0 items-center justify-center rounded-xl bg-sidebar-primary text-sidebar-primary-foreground">
          <UtensilsCrossed className="size-4.5" />
        </span>
        <div className="leading-tight">
          <p className="font-heading text-base font-semibold text-sidebar-foreground">MenuMind</p>
          <p className="text-xs text-sidebar-foreground/60">AI-Powered Menu Intelligence</p>
        </div>
      </div>

      <nav className="flex flex-1 flex-col gap-1 px-3">
        {NAV_ITEMS.map((item) => {
          const isActive = item.label === active
          const Icon = item.icon
          const itemClassName = cn(
            "flex items-center gap-3 rounded-xl px-3 py-2.5 text-sm font-medium transition-colors",
            isActive
              ? "bg-sidebar-primary text-sidebar-primary-foreground shadow-sm"
              : "text-sidebar-foreground/75 hover:bg-sidebar-accent hover:text-sidebar-accent-foreground"
          )

          if (item.href) {
            return (
              <Link key={item.label} href={item.href} className={itemClassName}>
                <Icon className="size-4.5" />
                {item.label}
              </Link>
            )
          }

          return (
            <div key={item.label} className={cn(itemClassName, "cursor-default opacity-60")}>
              <Icon className="size-4.5" />
              {item.label}
            </div>
          )
        })}
      </nav>

      <div className="mx-3 mb-4 flex items-center gap-3 rounded-xl border border-sidebar-border bg-sidebar-accent/40 px-3 py-2.5">
        <span className="flex size-9 shrink-0 items-center justify-center rounded-lg bg-sidebar-primary/20 text-sm font-semibold text-sidebar-primary">
          TT
        </span>
        <div className="min-w-0 leading-tight">
          <p className="truncate text-sm font-medium text-sidebar-foreground">Taco Town</p>
          <p className="truncate text-xs text-sidebar-foreground/60">Downtown Austin</p>
        </div>
      </div>
    </aside>
  )
}
