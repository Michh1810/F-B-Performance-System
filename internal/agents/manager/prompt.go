package manager

import (
	"fmt"
	"strings"
)

const systemPrompt = "You are the Manager Agent, the final decision-maker for a restaurant's menu. " +
	"You will be given a Trend Analysis (including structured TikTok metrics) and a Financial Analysis " +
	"for one menu item. Weigh the structured metrics directly, not just the narrative. Synthesize them " +
	"into a single decision. Respond with ONLY valid JSON matching this exact schema and nothing else: " +
	`{"decision": "LAUNCH" | "CUT" | "REPRICE", "reasoning": "<2-3 sentence justification>"}`

func buildPrompt(in Input) string {
	var b strings.Builder
	fmt.Fprintf(&b, "Menu item: %s\n\n", in.ItemName)

	b.WriteString("TikTok trend metrics:\n")
	fmt.Fprintf(&b, "- Mentions (video count): %d\n", in.TrendVideoCount)
	fmt.Fprintf(&b, "- Total views: %d\n", in.TrendTotalViews)
	fmt.Fprintf(&b, "- Engagement rate: %.2f%%\n", in.TrendEngagementRate*100)
	if in.TrendGrowthRatePct != nil {
		fmt.Fprintf(&b, "- Growth: %+.1f%% over %.1f hours\n", *in.TrendGrowthRatePct, in.TrendGrowthPeriodHours)
	} else {
		b.WriteString("- Growth: no baseline available yet\n")
	}
	fmt.Fprintf(&b, "- Sentiment: %s (score %.2f)\n", in.TrendSentimentLabel, in.TrendSentimentScore)
	if len(in.TrendRelatedFoodTrends) > 0 {
		fmt.Fprintf(&b, "- Related food trends: %s\n", strings.Join(in.TrendRelatedFoodTrends, ", "))
	}

	fmt.Fprintf(&b, "\nTrend Analysis (narrative):\n%s\n\nFinancial Analysis:\n%s", in.TrendAnalysis, in.FinancialAnalysis)
	return b.String()
}
