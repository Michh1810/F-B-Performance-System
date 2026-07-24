package trend

import (
	"fmt"
	"strings"
)

const systemPrompt = "You are a trend analyst for a restaurant product team. " +
	"You are given aggregated TikTok metrics for a menu item, already computed " +
	"from real video data (view counts, engagement, sentiment, growth, and " +
	"related food trends). Analyze them and determine the market viability of " +
	"the menu item. Be concise and specific. Do not invent numbers beyond what " +
	"is given."

// buildPrompt renders the latest snapshot's structured metrics, plus the
// growth computed against the prior snapshot, into a prompt for the
// Trend Agent's final narrative synthesis call.
func buildPrompt(in Input, latest Snapshot, growthPct *float64, growthPeriodHours float64) string {
	var b strings.Builder
	fmt.Fprintf(&b, "Menu item: %s\n\n", in.ItemName)
	fmt.Fprintf(&b, "TikTok snapshot captured: %s\n", latest.CapturedAt.Format("2006-01-02 15:04 MST"))
	fmt.Fprintf(&b, "Video count (mentions): %d\n", latest.VideoCount)
	fmt.Fprintf(&b, "Total views: %d\n", latest.TotalViews)
	fmt.Fprintf(&b, "Total likes: %d, comments: %d, shares: %d\n", latest.TotalLikes, latest.TotalComments, latest.TotalShares)
	fmt.Fprintf(&b, "Engagement rate: %.2f%%\n", latest.EngagementRate*100)
	fmt.Fprintf(&b, "Sentiment: %s (score %.2f, range -1 to 1)\n", latest.SentimentLabel, latest.SentimentScore)

	if growthPct != nil {
		fmt.Fprintf(&b, "Growth: %+.1f%% over the last %.1f hours\n", *growthPct, growthPeriodHours)
	} else if growthPeriodHours > 0 {
		fmt.Fprintf(&b, "Growth: no baseline to compare against (%.1f hours since prior snapshot)\n", growthPeriodHours)
	} else {
		fmt.Fprintf(&b, "Growth: no prior snapshot yet, this is the first data point\n")
	}

	if len(latest.TopHashtags) > 0 {
		fmt.Fprintf(&b, "Top hashtags: %s\n", strings.Join(latest.TopHashtags, ", "))
	}
	if len(latest.RelatedTrends) > 0 {
		fmt.Fprintf(&b, "Related food trends: %s\n", strings.Join(latest.RelatedTrends, ", "))
	}

	return b.String()
}

const snapshotSystemPrompt = "You classify TikTok food-trend signal from a sample of video captions " +
	"and hashtags. Respond ONLY with JSON matching this shape: " +
	`{"sentiment_label": "positive|neutral|negative", "sentiment_score": <number from -1 to 1>, ` +
	`"related_food_trends": ["..."]}. related_food_trends should be other food/menu trends these ` +
	"videos suggest are currently popular, inferred from captions and hashtags. Keep it to at most 5 items."

// buildSnapshotPrompt renders only the fields relevant to sentiment and
// related-trend classification (caption, hashtags, views) for a
// pre-selected subset of signals. It deliberately omits everything else
// (source, external IDs, timestamps) to keep the prompt compact and on-topic.
func buildSnapshotPrompt(signals []Signal) string {
	var b strings.Builder
	b.WriteString("TikTok videos semantically matching this item (top matches by view count):\n\n")
	for i, s := range signals {
		fmt.Fprintf(&b, "%d. (%d views) caption: %q hashtags: %s\n",
			i+1, s.ViewCount, s.Caption, strings.Join(s.Hashtags, ", "))
	}
	return b.String()
}
