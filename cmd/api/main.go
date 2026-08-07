package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/joho/godotenv"

	"fbperformance/internal/agents/financial"
	"fbperformance/internal/agents/hashtagsuggest"
	"fbperformance/internal/agents/manager"
	"fbperformance/internal/agents/menuidea"
	"fbperformance/internal/agents/orchestrator"
	"fbperformance/internal/agents/trend"
	"fbperformance/internal/ai"
	"fbperformance/internal/config"
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
	forecastingService := financial.NewServiceWithDB(db, aiClient)
	forecastingHandler := financial.NewHandler(forecastingService)
	menuItemStore := store.NewMenuItemStore(pool)

	llmClient := llm.NewClient(cfg.GeminiAPIKey)
	trendAgent := trend.NewAgent(llmClient, llmClient, signalStore, snapshotStore, cfg.GeminiModel, cfg.GeminiEmbedModel, cfg.TrendSignalLookbackDays)
	financialAgent := financial.NewAgent(llmClient, cfg.GeminiModel, menuItemStore, forecastingService)
	managerAgent := manager.NewAgent(llmClient, cfg.GeminiModel)
	recommendationOrchestrator := orchestrator.New(trendAgent, financialAgent, managerAgent)
	recommendationHandler := handlers.NewRecommendationHandler(recommendationOrchestrator)

	ideaStore := store.NewMenuIdeaStore(pool)
	menuIdeaCallLogStore := store.NewMenuIdeaCallLogStore(pool)
	menuIdeaAgent := menuidea.NewAgent(llmClient, llmClient, signalStore, cfg.GeminiModel, cfg.GeminiEmbedModel, cfg.TrendSignalLookbackDays)
	ideasHandler := handlers.NewIdeasHandler(ideaStore, menuItemStore, menuIdeaAgent, menuIdeaCallLogStore)

	restaurantProfileStore := store.NewRestaurantProfileStore(pool)
	restaurantProfileHandler := handlers.NewRestaurantProfileHandler(restaurantProfileStore)

	hashtagSuggestionStore := store.NewTrendHashtagSuggestionStore(pool)
	hashtagSuggestAgent := hashtagsuggest.NewAgent(llmClient, cfg.GeminiModel)
	trendHashtagsHandler := handlers.NewTrendHashtagsHandler(restaurantProfileStore, menuItemStore, hashtagSuggestAgent, hashtagSuggestionStore)

	menuItemsHandler := handlers.NewMenuItemsHandler(menuItemStore)

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
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   cfg.CORSAllowedOrigins,
		AllowedMethods:   []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Content-Type"},
		AllowCredentials: false,
		MaxAge:           300,
	}))

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
			r.Route("/ideas", func(r chi.Router) {
				r.Get("/", ideasHandler.List)
				r.Post("/run", ideasHandler.Run)
				r.Patch("/{id}", ideasHandler.UpdateStatus)
			})
		})

		r.Route("/menu-items", func(r chi.Router) {
			r.Get("/active", menuItemsHandler.ListActive)
		})

		r.Route("/restaurant-profile", func(r chi.Router) {
			r.Get("/", restaurantProfileHandler.Get)
			r.Put("/", restaurantProfileHandler.Upsert)
		})
		r.Route("/trend-hashtags/suggestions", func(r chi.Router) {
			r.Get("/", trendHashtagsHandler.List)
			r.Post("/", trendHashtagsHandler.Generate)
			r.Patch("/{id}", trendHashtagsHandler.UpdateStatus)
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
