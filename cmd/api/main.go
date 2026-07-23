package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"time"

	"fbperformance/internal/performance_analytics"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/robfig/cron/v3"
)

func main() {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		log.Fatal("DATABASE_URL is required")
	}

	db, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		log.Fatalf("failed to connect database: %v", err)
	}
	defer db.Close()

	repo := performance_analytics.NewRepository(db)
	service := performance_analytics.NewService(repo)
	handler := performance_analytics.NewHandler(service)

	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})
	mux.HandleFunc("/api/v1/dashboard/summary", handler.HandleSummary)
	mux.HandleFunc("/api/v1/dashboard/menu-items", handler.HandleMenuItems)
	mux.HandleFunc("/api/reviews", handler.ServeGoogleReviewHTTP)
	mux.HandleFunc("/api/clover", handler.ServeCloverOrdersHTTP)

	// --- CRON JOB SETUP ---
	c := cron.New()

	// Schedule to run clover_daily_seeder.go every day at 9:00 AM to populate the data for dashboard
	_, err = c.AddFunc("0 9 * * *", func() {
		log.Println("CRON: Running daily Clover seeder...")
		performance_analytics.SeedDailyCloverData()
	})
	if err != nil {
		log.Printf("Failed to schedule cron job: %v", err)
	} else {
		c.Start()
		defer c.Stop()
		log.Println("CRON: Scheduled daily Clover seeder at 9:00 AM")
	}
	// Schedule to extract reviews from Google Reviews at 9:00 AM daily
	_, err = c.AddFunc("0 9 * * *", func() {
		log.Println("CRON: Extracting daily Google reviews...")
		result, err := service.GetGoogleReviews()
		if err != nil {
			log.Printf("Failed to extract Google reviews: %v", err)
		}
		log.Printf("Extracted %d Google reviews", len(result.Reviews))
	})
	if err != nil {
		log.Printf("Failed to schedule cron job: %v", err)
	} else {
		c.Start()
		defer c.Stop()
		log.Println("CRON: Scheduled daily Google reviews at 9:00 AM")
	}
	// Test running google connection
	log.Println("TEST: Running GetGoogleReviews once on startup...")
	testResult, testErr := service.GetGoogleReviews()
	if testErr != nil {
		log.Printf("TEST Failed: %v", testErr)
	} else {
		log.Printf("TEST Success: Extracted %d Google reviews", len(testResult.Reviews))
	}
	// ----------------------
	log.Println("Server is running on port 8080...")
	log.Fatal(http.ListenAndServe(":8080", mux))
}
