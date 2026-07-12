package config

import "os"

// Config holds runtime configuration for the server, sourced from environment variables.
type Config struct {
	Port         string
	GeminiAPIKey string
	GeminiModel  string
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
	return Config{
		Port:         port,
		GeminiAPIKey: os.Getenv("GEMINI_API_KEY"),
		GeminiModel:  model,
	}
}
