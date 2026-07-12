// Command gemini-test is a scratch script for trying out prompts against
// Gemini directly, without starting the API server or crafting HTTP requests.
// Edit the prompt constant below and run: go run ./cmd/gemini-test
package main

import (
	"context"
	"fmt"
	"log"

	"github.com/joho/godotenv"

	"fbperformance/internal/config"
	"fbperformance/internal/gemini"
)

const prompt = "Suggest one way to increase restaurant table turnover."

func main() {
	_ = godotenv.Load()
	cfg := config.Load()

	client := gemini.NewClient(cfg.GeminiAPIKey)

	result, err := client.GenerateContent(context.Background(), cfg.GeminiModel, prompt)
	if err != nil {
		log.Fatalf("gemini call failed: %v", err)
	}

	fmt.Println(result)
}
