package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"time"

	"fbperformance/internal/performance_analytics"

	"github.com/jackc/pgx/v5/pgxpool"
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

	log.Println("Server is running on port 8080...")
	log.Fatal(http.ListenAndServe(":8080", mux))
}
