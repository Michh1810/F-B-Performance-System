package main

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"os"

	"fbperformance/internal/ai"
	"fbperformance/internal/demand_forecast"
	"fbperformance/internal/multiagent_recommendation"
	"fbperformance/internal/performance_analytics"

	_ "github.com/jackc/pgx/v5/stdlib"
)

func main() {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		dsn = "postgres://postgres:devpassword@localhost:5440/fbperformance?sslmode=disable"
	}

	db, err := sql.Open("pgx", dsn)
	if err != nil {
		log.Fatalf("failed to open database: %v", err)
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		log.Fatalf("failed to ping database: %v", err)
	}

	aiClient := ai.NewClientFromEnv()
	forecastingService := demand_forecast.NewServiceWithDB(db, aiClient)
	forecastingHandler := demand_forecast.NewHandler(forecastingService)
	recommendationService := multiagent_recommendation.NewService()
	recommendationHandler := multiagent_recommendation.NewHandler(recommendationService)
	analyticsService := performance_analytics.NewService()
	analyticsHandler := performance_analytics.NewHandler(analyticsService)

	http.Handle("/", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "Hello from Golang inside Docker LOL!")
	}))
	http.Handle("/api/forecast", forecastingHandler)
	http.Handle("/api/recommendations", recommendationHandler)
	http.Handle("/api/analytics", analyticsHandler)

	fmt.Println("Server is running on port 8080...")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
