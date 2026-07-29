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
		menuItemName: "Spicy Chicken Sandwich",
		signals: []mockSignal{
			{caption: "nashville hot chicken ramen bowl is the new mashup everyone's making", hashtags: []string{"foodtiktok", "nashvillehot", "ramen"}, viewCount: 820000, likeCount: 94000, commentCount: 2100, shareCount: 15000},
			{caption: "spicy chicken sando with a waffle bun instead of brioche, obsessed", hashtags: []string{"chickensando", "waffles", "foodtrends"}, viewCount: 410000, likeCount: 51000, commentCount: 1300, shareCount: 6200},
		},
	},
	{
		menuItemName: "Salted Egg Coffee",
		signals: []mockSignal{
			{caption: "salted egg yolk croissant filling is blowing up right now", hashtags: []string{"saltedegg", "croissant", "foodtiktok"}, viewCount: 610000, likeCount: 72000, commentCount: 1800, shareCount: 9000},
			{caption: "salted caramel cold brew layered with cream foam trend", hashtags: []string{"coffeetiktok", "saltedcaramel", "coldbrew"}, viewCount: 340000, likeCount: 39000, commentCount: 900, shareCount: 4100},
		},
	},
	{
		menuItemName: "Birria Tacos",
		signals: []mockSignal{
			{caption: "birria ramen fusion bowl, consomme as the broth, viral all over foodtok", hashtags: []string{"birria", "birriaramen", "foodtiktok"}, viewCount: 1200000, likeCount: 160000, commentCount: 4200, shareCount: 28000},
			{caption: "birria grilled cheese with consomme for dipping is unreal", hashtags: []string{"birria", "grilledcheese", "newmenuitem"}, viewCount: 780000, likeCount: 98000, commentCount: 2600, shareCount: 17000},
			// Names the item directly (not just an adjacent trend) —
			// exercises the promotion-opportunity classification, which
			// checks for the item's name in the caption text rather than a
			// distance sub-band (see mentionsItem in internal/agents/menuidea).
			{caption: "I'm shook!! Birria Tacos here are unreal, best in town", hashtags: []string{"birriatacos", "foodtiktok"}, viewCount: 2100000, likeCount: 310000, commentCount: 6800, shareCount: 45000},
		},
	},
	{
		menuItemName: "Matcha Latte",
		signals: []mockSignal{
			{caption: "matcha soft serve ice cream swirl trend at every cafe now", hashtags: []string{"matcha", "softserve", "foodtiktok"}, viewCount: 540000, likeCount: 61000, commentCount: 1400, shareCount: 7300},
			{caption: "strawberry matcha layered latte, the color split is so aesthetic", hashtags: []string{"matchalatte", "strawberrymatcha", "foodie"}, viewCount: 690000, likeCount: 88000, commentCount: 2000, shareCount: 12500},
		},
	},
	{
		menuItemName: "Loaded Nachos",
		signals: []mockSignal{
			{caption: "birria nachos with consomme drizzle is the new loaded nacho upgrade", hashtags: []string{"birria", "nachos", "foodtrends"}, viewCount: 470000, likeCount: 58000, commentCount: 1500, shareCount: 8100},
			{caption: "elote street corn nachos, cotija and chili lime everywhere", hashtags: []string{"elote", "nachos", "foodtiktok"}, viewCount: 310000, likeCount: 34000, commentCount: 800, shareCount: 3900},
		},
	},
	{
		menuItemName: "Korean Fried Chicken Wings",
		signals: []mockSignal{
			{caption: "gochujang wings glazed and double fried, viral korean fusion trend", hashtags: []string{"koreanfood", "gochujang", "friedchicken"}, viewCount: 560000, likeCount: 69000, commentCount: 1700, shareCount: 9800},
			{caption: "korean corn dog with cheese pull is still everywhere on foodtok", hashtags: []string{"koreancorndog", "foodtiktok", "cheesepull"}, viewCount: 920000, likeCount: 120000, commentCount: 3100, shareCount: 21000},
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
