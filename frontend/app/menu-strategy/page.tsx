import { Tabs, TabsContent } from "@/components/ui/tabs"
import { MenuActionPlan } from "@/components/menu-strategy/menu-action-plan"

export default function MenuStrategyPage() {
  return (
    <div className="flex min-h-svh flex-col p-6 max-w-5xl">
      <div className="mb-8">
        <h1 className="heading-large">Menu Strategy</h1>
      </div>

      <Tabs defaultValue="action-plan" className="w-full">
        <TabsContent value="action-plan">
          <MenuActionPlan />
        </TabsContent>
        <TabsContent value="simulator">
          <div className="rounded-lg border bg-card text-card-foreground shadow-sm p-6">
            <h2 className="heading-medium mb-4">Menu Simulator</h2>
            <p className="body-primary">Test Price & Demand impact tool will appear here.</p>
          </div>
        </TabsContent>
        <TabsContent value="agent-logic">
          <div className="rounded-lg border bg-card text-card-foreground shadow-sm p-6">
            <h2 className="heading-medium mb-4">Agent Logic</h2>
            <p className="body-primary">The &quot;Why&quot; behind the AI&apos;s choices will appear here.</p>
          </div>
        </TabsContent>
      </Tabs>
    </div>
  )
}
