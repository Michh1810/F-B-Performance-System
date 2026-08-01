import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs"

export default function SettingsPage() {
  return (
    <div className="flex min-h-svh flex-col p-6 max-w-5xl">
      <div className="mb-8">
        <h1 className="heading-large">Settings</h1>
        <p className="body-secondary mt-2">Configuration and Preferences</p>
      </div>

      <Tabs defaultValue="accounts" className="w-full">
        <TabsList className="mb-8">
          <TabsTrigger value="accounts">Connected Accounts</TabsTrigger>
          <TabsTrigger value="goals">Business Goals</TabsTrigger>
        </TabsList>
        <TabsContent value="accounts">
          <div className="rounded-lg border bg-card text-card-foreground shadow-sm p-6">
            <h2 className="heading-medium mb-4">Connected Accounts</h2>
            <p className="body-primary">Integrations for Clover, Google, and Yelp will appear here.</p>
          </div>
        </TabsContent>
        <TabsContent value="goals">
          <div className="rounded-lg border bg-card text-card-foreground shadow-sm p-6">
            <h2 className="heading-medium mb-4">Business Goals</h2>
            <p className="body-primary">Target margins and AI risk tolerance configurations will appear here.</p>
          </div>
        </TabsContent>
      </Tabs>
    </div>
  )
}
