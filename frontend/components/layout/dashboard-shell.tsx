import { AppSidebar } from "@/components/layout/app-sidebar"

export function DashboardShell({
  active,
  children,
}: {
  active: string
  children: React.ReactNode
}) {
  return (
    <div className="flex min-h-svh bg-background">
      <AppSidebar active={active} />
      <main className="min-w-0 flex-1 overflow-x-hidden">{children}</main>
    </div>
  )
}
