export const criticalAlert = {
  active: true,
  type: "warning" as const, // or "destructive"
  title: "Action Required: Revenue Drop",
  message: "Revenue from your top 3 sushi rolls dropped 20% this week compared to the 30-day average. View Performance Data for details."
}

export const weeklyPulseData = {
  revenue: { value: 34250, label: "Total Weekly Revenue", trend: "+5%", trendUp: true, vs: "previous week" },
  aov: { value: 42.15, label: "Average Order Value", trend: "+2.1%", trendUp: true, vs: "previous week" },
  orders: { value: 812, label: "Total Orders", trend: "-1.5%", trendUp: false, vs: "previous week" },
  sentiment: { value: 4.8, label: "Overall Sentiment", trend: "+0.2", trendUp: true, vs: "previous week (24 new reviews)" },
}

export const aiInsightsData = [
  {
    id: "marketing",
    pillar: "Marketing & Promotions",
    title: "Driving Demand",
    icon: "trendingUp",
    suggestion: "Marketing Opportunity: 'Spicy Tuna Roll' sales volume dropped 15% this week despite steady overall foot traffic. Consider making it the centerpiece of this week's Instagram promo."
  },
  {
    id: "pricing",
    pillar: "Menu Strategy & Pricing",
    title: "Price Elasticity",
    icon: "dollarSign",
    suggestion: "Pricing Insight: You raised the price of 'Dragon Roll' by $1.00 two weeks ago, and sales volume has remained completely flat. The market accepted the price increase."
  },
  {
    id: "operations",
    pillar: "Operations",
    title: "POS Friction & Errors",
    icon: "alertCircle",
    suggestion: "Operational Warning: Voids and comped items spiked by 25% on Friday night between 6 PM and 8 PM. Check in with the Friday kitchen staff regarding order accuracy."
  },
  {
    id: "sentiment",
    pillar: "Guest Sentiment",
    title: "Reputation Management",
    icon: "messageSquare",
    suggestion: "Service Insight: 3 reviews this week praised the 'new bartender' but mentioned the 'music was too loud'. Adjust the dining room audio mix for the evening shifts."
  }
]
