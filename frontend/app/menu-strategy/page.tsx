import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs"

export default function MenuStrategyPage() {
  return (
    <div className="flex min-h-svh flex-col p-6 max-w-5xl">
      <div className="mb-8">
        <h1 className="heading-large">Menu Strategy</h1>
        <p className="body-secondary mt-2">The Action Center: AI & Forecasting</p>
      </div>

      <Tabs defaultValue="action-plan" className="w-full">
        <TabsList className="mb-8">
          <TabsTrigger value="action-plan">AI Action Plan</TabsTrigger>
          <TabsTrigger value="simulator">Menu Simulator</TabsTrigger>
          <TabsTrigger value="agent-logic">Agent Logic</TabsTrigger>
        </TabsList>
        <TabsContent value="action-plan">
          <div className="rounded-lg border bg-card text-card-foreground shadow-sm p-6">
            <h2 className="heading-medium mb-4">AI Action Plan</h2>
            <p className="body-primary">Launch, Improve, Drop, Reprice recommendations will appear here.</p>
          </div>
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
            <p className="body-primary">The "Why" behind the AI's choices will appear here.</p>
          </div>
        </TabsContent>
      </Tabs>
    </div>
  )
}
