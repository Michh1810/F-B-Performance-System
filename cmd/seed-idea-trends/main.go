// Command seed-idea-trends is a one-shot dev-data seeder for the Menu Idea
// Agent (internal/agents/menuidea): for each menu item cmd/seed-dev creates,
// it authors a few curated TikTok captions describing trends that are
// topically adjacent to that item (not the item itself), embeds each one
// via the real Gemini embedding API, and upserts them into trend_signals
// under source "mock". This lets cmd/menu-idea-gen be run locally and
// produce real, meaningful ideas without needing an Apify token or waiting
// on cmd/trend-ingest to sweep real TikTok content.
//
// It is safe to re-run: TrendSignalStore.Upsert is keyed on (source,
// external_id), so re-running just refreshes engagement counters rather
// than duplicating rows or re-embedding captions.
//
// Requires GEMINI_API_KEY (and DATABASE_URL, if not the local default) to
// be set — this seeder calls the real embedding API, it does not fabricate
// vectors.
package main

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/joho/godotenv"

	"fbperformance/internal/agents/trend"
	"fbperformance/internal/config"
	"fbperformance/internal/services/llm"
	"fbperformance/internal/store"
)

// mockSignal is one authored TikTok caption to embed and upsert.
type mockSignal struct {
	caption      string
	hashtags     []string
	viewCount    int64
	likeCount    int64
	commentCount int64
	shareCount   int64
}

// seedGroup pairs a menu item (must match a name seeded by cmd/seed-dev)
// with captions describing trends adjacent to — but distinct from — that
// item, so SearchAdjacent's band has something real to find.
type seedGroup struct {
	menuItemName string
	signals      []mockSignal
}

var seedGroups = []seedGroup{
	{
		menuItemName: "Bulgogi Bowl",
		signals: []mockSignal{
			{caption: "korean bbq bowls with a fried egg on top are everywhere on foodtok right now", hashtags: []string{"koreanbbq", "ricebowl", "foodtiktok"}, viewCount: 820000, likeCount: 94000, commentCount: 2100, shareCount: 15000},
			{caption: "gochujang glazed beef bowls layered over rice, the sauce drip shots are unreal", hashtags: []string{"gochujang", "beefbowl", "foodtrends"}, viewCount: 410000, likeCount: 51000, commentCount: 1300, shareCount: 6200},
		},
	},
	{
		menuItemName: "Kimchi Fried Rice",
		signals: []mockSignal{
			{caption: "kimchi fried rice topped with a crispy fried egg and nori strips trending hard", hashtags: []string{"kimchi", "friedrice", "foodtiktok"}, viewCount: 610000, likeCount: 72000, commentCount: 1800, shareCount: 9000},
			{caption: "spam and kimchi rice bowls with a soy butter drizzle blowing up", hashtags: []string{"spam", "kimchirice", "koreanfood"}, viewCount: 340000, likeCount: 39000, commentCount: 900, shareCount: 4100},
		},
	},
	{
		menuItemName: "Bibimbap",
		signals: []mockSignal{
			{caption: "stone bowl bibimbap with the sizzling rice crust everyone's obsessed with", hashtags: []string{"bibimbap", "stonebowl", "foodtiktok"}, viewCount: 1200000, likeCount: 160000, commentCount: 4200, shareCount: 28000},
			{caption: "vegan bibimbap bowls with gochujang tofu are the new foodtok staple", hashtags: []string{"vegan", "bibimbap", "koreanfood"}, viewCount: 780000, likeCount: 98000, commentCount: 2600, shareCount: 17000},
			// Names the item directly (not just an adjacent trend) —
			// exercises the promotion-opportunity classification, which
			// checks for the item's name in the caption text rather than a
			// distance sub-band (see mentionsItem in internal/agents/menuidea).
			{caption: "I'm shook!! This Bibimbap here is unreal, best in town", hashtags: []string{"bibimbap", "foodtiktok"}, viewCount: 2100000, likeCount: 310000, commentCount: 6800, shareCount: 45000},
		},
	},
	{
		menuItemName: "Jeyuk Bokkeum",
		signals: []mockSignal{
			{caption: "spicy korean pork belly stir fry lettuce wraps trending all over tiktok", hashtags: []string{"koreanbbq", "porkbelly", "foodtiktok"}, viewCount: 540000, likeCount: 61000, commentCount: 1400, shareCount: 7300},
			{caption: "gochujang glazed pork skewers grilled tableside, the char marks are insane", hashtags: []string{"gochujang", "porkskewers", "foodie"}, viewCount: 690000, likeCount: 88000, commentCount: 2000, shareCount: 12500},
		},
	},
	{
		menuItemName: "Yangnyeom Fried Chicken",
		signals: []mockSignal{
			{caption: "double fried korean chicken tossed in sweet spicy glaze, the crunch asmr videos are everywhere", hashtags: []string{"koreanfriedchicken", "asmr", "foodtiktok"}, viewCount: 470000, likeCount: 58000, commentCount: 1500, shareCount: 8100},
			{caption: "korean fried chicken sandwiches with pickled radish are the new sando trend", hashtags: []string{"friedchickensando", "koreanfood", "foodtrends"}, viewCount: 310000, likeCount: 34000, commentCount: 800, shareCount: 3900},
		},
	},
	{
		menuItemName: "Japchae",
		signals: []mockSignal{
			{caption: "glass noodle stir fry bowls with sesame oil going viral on foodtok", hashtags: []string{"japchae", "glassnoodles", "foodtiktok"}, viewCount: 560000, likeCount: 69000, commentCount: 1700, shareCount: 9800},
			{caption: "japchae carbonara fusion noodles, the cream and gochugaru combo is wild", hashtags: []string{"japchae", "fusionfood", "noodles"}, viewCount: 290000, likeCount: 31000, commentCount: 700, shareCount: 3400},
		},
	},
	{
		menuItemName: "Kimchijeon",
		signals: []mockSignal{
			{caption: "crispy kimchi pancakes with a soy vinegar dip, the crackle sound videos are everywhere", hashtags: []string{"kimchijeon", "koreanpancake", "asmr"}, viewCount: 380000, likeCount: 42000, commentCount: 1000, shareCount: 5200},
			{caption: "scallion and kimchi savory pancakes stacked tall, viral brunch trend", hashtags: []string{"pajeon", "koreanbrunch", "foodtiktok"}, viewCount: 260000, likeCount: 27000, commentCount: 650, shareCount: 3000},
		},
	},
	{
		menuItemName: "Mandu Dumplings",
		signals: []mockSignal{
			{caption: "pan fried dumplings with a lacy crispy skirt, the frico dumpling trend is everywhere", hashtags: []string{"mandu", "dumplings", "foodtiktok"}, viewCount: 450000, likeCount: 53000, commentCount: 1300, shareCount: 7000},
			{caption: "steamed mandu dipped in chili oil, the fold technique videos are blowing up", hashtags: []string{"mandu", "koreanfood", "dumplings"}, viewCount: 310000, likeCount: 34000, commentCount: 850, shareCount: 3800},
		},
	},
	{
		menuItemName: "Tteokbokki",
		signals: []mockSignal{
			{caption: "cheese tteokbokki with mozzarella pull shots going viral again", hashtags: []string{"tteokbokki", "cheesepull", "foodtiktok"}, viewCount: 920000, likeCount: 120000, commentCount: 3100, shareCount: 21000},
			{caption: "rosé tteokbokki, the creamy gochujang sauce trend is taking over foodtok", hashtags: []string{"rosetteokbokki", "gochujang", "koreanfood"}, viewCount: 500000, likeCount: 60000, commentCount: 1600, shareCount: 8600},
		},
	},
	{
		menuItemName: "Korean Corn Dog",
		signals: []mockSignal{
			{caption: "half rice cake half mozzarella corn dogs rolled in sugar and ramen crumbs, viral street food", hashtags: []string{"koreancorndog", "streetfood", "foodtiktok"}, viewCount: 1400000, likeCount: 210000, commentCount: 5200, shareCount: 33000},
			{caption: "double cheese korean corn dogs with the cheese pull close-up shots everywhere", hashtags: []string{"koreancorndog", "cheesepull", "foodtrends"}, viewCount: 860000, likeCount: 110000, commentCount: 2900, shareCount: 19000},
			{caption: "this Korean Corn Dog stand had the longest line, worth every minute", hashtags: []string{"koreancorndog", "foodtiktok"}, viewCount: 1900000, likeCount: 280000, commentCount: 6100, shareCount: 40000},
		},
	},
	{
		menuItemName: "Yuja Citron Tea",
		signals: []mockSignal{
			{caption: "citrus honey tea poured over ice with whole fruit slices, aesthetic drink trend", hashtags: []string{"yujacha", "citrustea", "foodtiktok"}, viewCount: 240000, likeCount: 26000, commentCount: 600, shareCount: 2800},
			{caption: "yuzu soda floats with sparkling water are the new cafe drink everyone's making", hashtags: []string{"yuzu", "cafedrinks", "foodtrends"}, viewCount: 300000, likeCount: 33000, commentCount: 750, shareCount: 3600},
		},
	},
	{
		menuItemName: "Sikhye",
		signals: []mockSignal{
			{caption: "korean rice punch served ice cold with floating rice grains, nostalgic drink trend", hashtags: []string{"sikhye", "koreandrinks", "foodtiktok"}, viewCount: 180000, likeCount: 19000, commentCount: 450, shareCount: 2000},
			{caption: "sweet rice drinks in glass bottles going viral as a dessert-table staple", hashtags: []string{"koreanfood", "sweetdrinks", "foodie"}, viewCount: 150000, likeCount: 15000, commentCount: 380, shareCount: 1600},
		},
	},
	{
		menuItemName: "Injeolmi Bingsu",
		signals: []mockSignal{
			{caption: "shaved milk ice topped with roasted soybean powder and rice cakes, viral dessert", hashtags: []string{"bingsu", "injeolmi", "foodtiktok"}, viewCount: 670000, likeCount: 81000, commentCount: 2000, shareCount: 11000},
			{caption: "matcha injeolmi bingsu bowls shared between friends, the group dessert trend", hashtags: []string{"bingsu", "matcha", "koreandessert"}, viewCount: 420000, likeCount: 47000, commentCount: 1100, shareCount: 6300},
		},
	},
}

func main() {
	_ = godotenv.Load()

	cfg := config.Load()
	if cfg.GeminiAPIKey == "" {
		log.Fatal("seed-idea-trends: GEMINI_API_KEY is required (this seeder embeds captions via the real Gemini API)")
	}
	ctx := context.Background()

	pool, err := store.Connect(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("connect database: %v", err)
	}
	defer pool.Close()

	signalStore := store.NewTrendSignalStore(pool)
	llmClient := llm.NewClient(cfg.GeminiAPIKey)
	now := time.Now().UTC()

	var upserted int
	for _, group := range seedGroups {
		for i, sig := range group.signals {
			embedding, err := llmClient.Embed(ctx, cfg.GeminiEmbedModel, sig.caption, llm.EmbedOptions{
				TaskType:             "RETRIEVAL_DOCUMENT",
				OutputDimensionality: trend.EmbeddingDimensions,
			})
			if err != nil {
				log.Printf("seed-idea-trends: %s: embed caption %d: %v", group.menuItemName, i, err)
				continue
			}

			externalID := fmt.Sprintf("%s-%d", slugify(group.menuItemName), i)
			signal := trend.Signal{
				Source:       "mock",
				ExternalID:   externalID,
				Caption:      sig.caption,
				Hashtags:     sig.hashtags,
				ViewCount:    sig.viewCount,
				LikeCount:    sig.likeCount,
				CommentCount: sig.commentCount,
				ShareCount:   sig.shareCount,
				PostedAt:     now.Add(-time.Duration(i+1) * 6 * time.Hour),
				URL:          fmt.Sprintf("https://www.tiktok.com/@mockuser/video/%s", externalID),
			}
			if err := signalStore.Upsert(ctx, signal, embedding); err != nil {
				log.Printf("seed-idea-trends: %s: upsert caption %d: %v", group.menuItemName, i, err)
				continue
			}
			upserted++
			log.Printf("seed-idea-trends: %-28s adjacent signal %d/%d upserted", group.menuItemName, i+1, len(group.signals))
		}
	}

	log.Printf("seed-idea-trends: done, %d signals upserted (source=mock). Run cmd/menu-idea-gen next, or GET /api/ai/ideas after.", upserted)
}

// slugify lowercases a menu item name and replaces spaces with hyphens, for
// a stable, readable external_id namespace per item.
func slugify(name string) string {
	return strings.ToLower(strings.ReplaceAll(name, " ", "-"))
}
