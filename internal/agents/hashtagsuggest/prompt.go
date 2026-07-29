package hashtagsuggest

import (
	"fmt"
	"strings"
)

const systemPrompt = "You are a social-media strategist for a restaurant, picking TikTok hashtags " +
	"for a trend-scraping job to sweep. Given the restaurant's description and current menu, suggest " +
	"hashtags likely to surface TikTok content relevant to trends adjacent to this restaurant's food " +
	"— not just hashtags describing the exact dishes it already sells. Prefer hashtags with real TikTok " +
	"content volume (broad cuisine/format tags like \"birriatacos\" or \"koreanfriedchicken\") over " +
	"hyper-specific or invented ones. Respond ONLY with JSON matching this shape: " +
	`{"hashtags": ["...", "..."]}. Suggest between 6 and 12 hashtags, lowercase, no "#" symbol.`

// buildPrompt renders the restaurant description plus a compact list of
// current active menu items (name + category) for grounding.
func buildPrompt(in Input) string {
	var b strings.Builder
	fmt.Fprintf(&b, "Restaurant description: %s\n\n", in.Description)
	if len(in.MenuItems) == 0 {
		b.WriteString("Current menu: (no active menu items yet)\n")
		return b.String()
	}
	b.WriteString("Current menu items:\n")
	for _, item := range in.MenuItems {
		fmt.Fprintf(&b, "- %s (%s)\n", item.Name, item.Category)
	}
	return b.String()
}
