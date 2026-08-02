package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/joho/godotenv"

	"fbperformance/internal/agents/financial"
	"fbperformance/internal/agents/manager"
	"fbperformance/internal/agents/orchestrator"
	"fbperformance/internal/agents/trend"
	"fbperformance/internal/ai"
	"fbperformance/internal/config"
	"fbperformance/internal/demand_forecast"
	"fbperformance/internal/handlers"
	"fbperformance/internal/overview"
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

	logDatabaseTarget(databaseURL)

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

	store.RunMigrations(databaseURL)

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

	repo := performance_analytics.NewRepository(pool)
	analyticsService := performance_analytics.NewService(repo)
	analyticsHandler := performance_analytics.NewHandler(analyticsService)

	overviewRepo := overview.NewRepository(pool)
	overviewService := overview.NewService(overviewRepo, llmClient)
	overviewHandler := overview.NewHandler(overviewService)

	port := cfg.Port
	if port == "" {
		port = os.Getenv("PORT")
	}
	if port == "" {
		port = "8080"
	}

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
	r.Get("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})

	r.Get("/api/analytics", analyticsHandler.HandleSummary)

	r.Route("/api", func(r chi.Router) {
		r.Handle("/forecast", forecastingHandler)
		r.Route("/ai", func(r chi.Router) {
			r.Handle("/recommendation", recommendationHandler)
		})

		r.Get("/v1/dashboard/summary", analyticsHandler.HandleSummary)
		r.Get("/v1/dashboard/menu-items", analyticsHandler.HandleMenuItems)
		r.Get("/v1/performance-dashboard", analyticsHandler.HandlePerformanceDashboard)
		r.Get("/v1/overview", overviewHandler.HandleGetOverview)
		r.Get("/reviews", analyticsHandler.ServeGoogleReviewHTTP)
		r.Get("/clover", analyticsHandler.ServeCloverOrdersHTTP)
	})

	log.Printf("listening on :%s", port)
	log.Fatal(http.ListenAndServe(":"+port, r))
}

func logDatabaseTarget(databaseURL string) {
	u, err := url.Parse(databaseURL)
	if err != nil {
		log.Printf("database target: unable to parse DATABASE_URL: %v", err)
		return
	}

	host := u.Hostname()
	dbName := strings.TrimPrefix(u.Path, "/")
	if dbName == "" {
		dbName = "(empty)"
	}

	log.Printf("database target: host=%s db=%s", host, dbName)
}
