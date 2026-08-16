package menuidea

import (
	"fmt"
	"sort"
	"strings"
)

const systemPrompt = "You are a menu innovation analyst for a restaurant. You are given " +
	"an existing menu item and a sample of TikTok videos that are topically adjacent to " +
	"it - related food trends, not videos about this exact item. Propose 0 to 2 concrete " +
	"new menu items that would let the restaurant capitalize on this trend, each clearly " +
	"distinct from the existing item (not a rename or minor variant of it). Ground every " +
	"idea only in the evidence given - do not invent trends not present in the videos. " +
	"If nothing in the evidence supports a compelling new item, return an empty list. " +
	`Respond ONLY with JSON matching this shape: {"ideas": [{"name": "...", ` +
	`"description": "...", "suggested_category": "...", "rationale": "..."}]}`

// maxPromptSignals bounds how many tweak-candidate signals are rendered
// into the prompt, keeping it compact and cheap even when SearchRelevant
// returns many matches.
const maxPromptSignals = 10

// buildIdeaPrompt renders the existing menu item plus its top
// tweak-candidate signals (by view count, already filtered to exclude
// ones naming the item — see splitByMention) for the idea-generation call.
func buildIdeaPrompt(item MenuItem, signals []RelevantSignal) string {
	var b strings.Builder
	fmt.Fprintf(&b, "Existing menu item: %s (category: %s)\n\n", item.Name, item.Category)
	b.WriteString("TikTok videos adjacent to this item (not about it directly), top matches by view count:\n\n")
	for i, s := range selectTopAdjacentSignals(signals, maxPromptSignals) {
		fmt.Fprintf(&b, "%d. (%d views) caption: %q hashtags: %s\n",
			i+1, s.ViewCount, s.Caption, strings.Join(s.Hashtags, ", "))
	}
	return b.String()
}

// selectTopAdjacentSignals returns up to n signals sorted by view count
// descending, without mutating the order of the caller's slice.
func selectTopAdjacentSignals(signals []RelevantSignal, n int) []RelevantSignal {
	sorted := make([]RelevantSignal, len(signals))
	copy(sorted, signals)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i].ViewCount > sorted[j].ViewCount })
	if len(sorted) > n {
		sorted = sorted[:n]
	}
	return sorted
}
