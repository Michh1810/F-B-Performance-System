package config

import (
	"os"
	"strconv"
	"strings"
)

// defaultTrendIngestHashtags is the fallback set of broad, menu-item-agnostic
// food-trend hashtags cmd/trend-ingest sweeps when TREND_INGEST_HASHTAGS is unset.
var defaultTrendIngestHashtags = []string{"foodtiktok", "foodtrends", "foodreview", "newmenuitem", "foodie"}

// Config holds runtime configuration for the server, sourced from environment variables.
type Config struct {
	Port             string
	GeminiAPIKey     string
	GeminiModel      string
	GeminiEmbedModel string

	DatabaseURL string

	ApifyAPIToken string
	ApifyActorID  string

	TrendSignalLookbackDays int
	TrendIngestHashtags     []string
}

func Load() Config {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	model := os.Getenv("GEMINI_MODEL")
	if model == "" {
		model = "gemini-flash-latest"
	}
	embedModel := os.Getenv("GEMINI_EMBED_MODEL")
	if embedModel == "" {
		embedModel = "gemini-embedding-001"
	}
	actorID := os.Getenv("APIFY_ACTOR_ID")
	if actorID == "" {
		actorID = "clockworks~tiktok-scraper"
	}

	lookbackDays := 30
	if v := os.Getenv("TREND_SIGNAL_LOOKBACK_DAYS"); v != "" {
		if parsed, err := strconv.Atoi(v); err == nil {
			lookbackDays = parsed
		}
	}

	hashtags := defaultTrendIngestHashtags
	if v := os.Getenv("TREND_INGEST_HASHTAGS"); v != "" {
		hashtags = strings.Split(v, ",")
		for i := range hashtags {
			hashtags[i] = strings.TrimSpace(hashtags[i])
		}
	}

	return Config{
		Port:             port,
		GeminiAPIKey:     os.Getenv("GEMINI_API_KEY"),
		GeminiModel:      model,
		GeminiEmbedModel: embedModel,

		DatabaseURL: os.Getenv("DATABASE_URL"),

		ApifyAPIToken: os.Getenv("APIFY_API_TOKEN"),
		ApifyActorID:  actorID,

		TrendSignalLookbackDays: lookbackDays,
		TrendIngestHashtags:     hashtags,
	}
}
