package main

import (
	"encoding/json"
	"log"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/joho/godotenv"

	"fbperformance/internal/config"
	"fbperformance/internal/gemini"
	"fbperformance/internal/multiagent_recommendation"
	"fbperformance/internal/performance_analytics"
)

func main() {
	// .env is optional; ignore the error if it's not present (e.g. in Docker
	// where vars are set directly via docker-compose).
	_ = godotenv.Load()

	cfg := config.Load()

	geminiClient := gemini.NewClient(cfg.GeminiAPIKey)
	recommendationService := multiagent_recommendation.NewService(geminiClient, cfg.GeminiModel)
	recommendationHandler := multiagent_recommendation.NewHandler(recommendationService)

	analyticsService := performance_analytics.NewService()
	analyticsHandler := performance_analytics.NewHandler(analyticsService)

	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Timeout(60 * time.Second))

	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
	})

	r.Route("/api", func(r chi.Router) {
		r.Handle("/performance-analytics", analyticsHandler)

		r.Route("/ai", func(r chi.Router) {
			r.Handle("/recommendation", recommendationHandler)
		})
	})

	log.Printf("listening on :%s", cfg.Port)
	log.Fatal(http.ListenAndServe(":"+cfg.Port, r))
}
