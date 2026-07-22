package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/joho/godotenv"

	"fbperformance/internal/agents/financial"
	"fbperformance/internal/agents/manager"
	"fbperformance/internal/agents/orchestrator"
	"fbperformance/internal/agents/trend"
	"fbperformance/internal/config"
	"fbperformance/internal/handlers"
	"fbperformance/internal/performance_analytics"
	"fbperformance/internal/services/llm"
	"fbperformance/internal/store"
)

func main() {
	// .env is optional; ignore the error if it's not present (e.g. in Docker
	// where vars are set directly via docker-compose).
	_ = godotenv.Load()

	cfg := config.Load()

	pool, err := store.Connect(context.Background(), cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("connect database: %v", err)
	}
	defer pool.Close()
	snapshotStore := store.NewTrendSnapshotStore(pool)
	signalStore := store.NewTrendSignalStore(pool)

	llmClient := llm.NewClient(cfg.GeminiAPIKey)
	trendAgent := trend.NewAgent(llmClient, llmClient, signalStore, snapshotStore, cfg.GeminiModel, cfg.GeminiEmbedModel, cfg.TrendSignalLookbackDays)
	financialAgent := financial.NewAgent(llmClient, cfg.GeminiModel)
	managerAgent := manager.NewAgent(llmClient, cfg.GeminiModel)
	recommendationOrchestrator := orchestrator.New(trendAgent, financialAgent, managerAgent)
	recommendationHandler := handlers.NewRecommendationHandler(recommendationOrchestrator)

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
