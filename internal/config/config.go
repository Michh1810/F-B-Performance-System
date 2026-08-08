package config

import (
	"os"
	"strconv"
	"strings"
	"time"
)

// defaultTrendIngestHashtags is the fallback set of broad, menu-item-agnostic
// food-trend hashtags cmd/trend-ingest sweeps when TREND_INGEST_HASHTAGS is unset.
var defaultTrendIngestHashtags = []string{"foodtiktok", "foodtrends", "foodreview", "newmenuitem", "foodie"}

// weekdaysByName maps TREND_INGEST_SCHEDULE_WEEKDAY's accepted values
// (case-insensitive full day names) to time.Weekday.
var weekdaysByName = map[string]time.Weekday{
	"sunday":    time.Sunday,
	"monday":    time.Monday,
	"tuesday":   time.Tuesday,
	"wednesday": time.Wednesday,
	"thursday":  time.Thursday,
	"friday":    time.Friday,
	"saturday":  time.Saturday,
}

// Config holds runtime configuration for the server, sourced from environment variables.
type Config struct {
	Port             string
	GeminiAPIKey     string
	GeminiModel      string
	GeminiEmbedModel string

	PostHogAPIKey               string
	PostHogHost                 string
	PostHogDistinctID           string
	PostHogFeatureFlagTimeoutMS int

	DatabaseURL string

	ApifyAPIToken string
	ApifyActorID  string

	TrendSignalLookbackDays int
	TrendIngestHashtags     []string

	// TrendIngestScheduleWeekday/Hour control the API server's automatic
	// weekly re-sweep (see internal/services/trendingest.Scheduler) — an
	// addition on top of cmd/trend-ingest itself, which remains a
	// non-self-scheduling CLI (see its doc comment). Hour is UTC, 0-23.
	TrendIngestScheduleWeekday time.Weekday
	TrendIngestScheduleHour    int

	// CORSAllowedOrigins is who may call the API from a browser (the
	// frontend's dev/prod origins) — see cmd/api/main.go's cors.Handler.
	CORSAllowedOrigins []string
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

	scheduleWeekday := time.Sunday
	if v := os.Getenv("TREND_INGEST_SCHEDULE_WEEKDAY"); v != "" {
		if parsed, ok := weekdaysByName[strings.ToLower(strings.TrimSpace(v))]; ok {
			scheduleWeekday = parsed
		}
	}
	scheduleHour := 3
	if v := os.Getenv("TREND_INGEST_SCHEDULE_HOUR"); v != "" {
		if parsed, err := strconv.Atoi(v); err == nil && parsed >= 0 && parsed <= 23 {
			scheduleHour = parsed
		}
	}

	corsOrigins := []string{"http://localhost:3000"}
	if v := os.Getenv("CORS_ALLOWED_ORIGINS"); v != "" {
		corsOrigins = strings.Split(v, ",")
		for i := range corsOrigins {
			corsOrigins[i] = strings.TrimSpace(corsOrigins[i])
		}
	}

	postHogDistinctID := os.Getenv("POSTHOG_DISTINCT_ID")
	if postHogDistinctID == "" {
		postHogDistinctID = "fbperformance-backend"
	}

	postHogTimeoutMS := 200
	if v := os.Getenv("POSTHOG_FEATURE_FLAG_TIMEOUT_MS"); v != "" {
		if parsed, err := strconv.Atoi(v); err == nil && parsed > 0 {
			postHogTimeoutMS = parsed
		}
	}

	return Config{
		Port:             port,
		GeminiAPIKey:     os.Getenv("GEMINI_API_KEY"),
		GeminiModel:      model,
		GeminiEmbedModel: embedModel,

		PostHogAPIKey:               os.Getenv("POSTHOG_API_KEY"),
		PostHogHost:                 os.Getenv("POSTHOG_HOST"),
		PostHogDistinctID:           postHogDistinctID,
		PostHogFeatureFlagTimeoutMS: postHogTimeoutMS,

		DatabaseURL: os.Getenv("DATABASE_URL"),

		ApifyAPIToken: os.Getenv("APIFY_API_TOKEN"),
		ApifyActorID:  actorID,

		TrendSignalLookbackDays: lookbackDays,
		TrendIngestHashtags:     hashtags,

		TrendIngestScheduleWeekday: scheduleWeekday,
		TrendIngestScheduleHour:    scheduleHour,

		CORSAllowedOrigins: corsOrigins,
	}
}
