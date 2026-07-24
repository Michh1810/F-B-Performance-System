package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"log"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/joho/godotenv"
	_ "github.com/jackc/pgx/v5/stdlib"

	"fbperformance/internal/agents/financial"
	"fbperformance/internal/agents/manager"
	"fbperformance/internal/agents/orchestrator"
	"fbperformance/internal/agents/trend"
	"fbperformance/internal/ai"
	"fbperformance/internal/config"
	"fbperformance/internal/demand_forecast"
	"fbperformance/internal/handlers"
	"fbperformance/internal/performance_analytics"
	"fbperformance/internal/services/llm"
	"fbperformance/internal/store"
)

func main() {
	_ = godotenv.Load()

	cfg := config.Load()

	databaseURL := cfg.DatabaseURL
	if databaseURL == "" {
		databaseURL = "postgres://postgres:devpassword@localhost:5440/fbperformance?sslmode=disable"
	}

	db, err := sql.Open("pgx", databaseURL)
	if err != nil {
		log.Fatalf("failed to open database: %v", err)
	}
	defer db.Close()
	if err := db.Ping(); err != nil {
		log.Fatalf("failed to ping database: %v", err)
	}

	pool, err := store.Connect(context.Background(), databaseURL)
	if err != nil {
		log.Fatalf("connect database: %v", err)
	}
	defer pool.Close()

	snapshotStore := store.NewTrendSnapshotStore(pool)
	signalStore := store.NewTrendSignalStore(pool)

	aiClient := ai.NewClientFromEnv()
	forecastingService := demand_forecast.NewServiceWithDB(db, aiClient)
	forecastingHandler := demand_forecast.NewHandler(forecastingService)

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
		_ = json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
	})

	r.Handle("/api/recommendations", recommendationHandler)
	r.Handle("/api/analytics", analyticsHandler)

	r.Route("/api", func(r chi.Router) {
		r.Handle("/performance-analytics", analyticsHandler)
		r.Handle("/forecast", forecastingHandler)
		r.Route("/ai", func(r chi.Router) {
			r.Handle("/recommendation", recommendationHandler)
		})
	})

	log.Printf("listening on :%s", cfg.Port)
	log.Fatal(http.ListenAndServe(":"+cfg.Port, r))
}
