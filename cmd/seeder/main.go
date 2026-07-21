package main

import (
	"bufio"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"os"

	_ "github.com/lib/pq"
)

type YelpReviewJSON struct {
	ReviewID   string  `json:"review_id"`
	UserID     string  `json:"user_id"`
	BusinessID string  `json:"business_id"`
	Stars      float64 `json:"stars"`
	Date       string  `json:"date"`
	Text       string  `json:"text"`
}

func main() {
	db, err := sql.Open("postgres", os.Getenv("DATABASE_URL"))
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	file, err := os.Open("sample_json_response/yelp_academic_dataset_review.json")
	if err != nil {
		log.Fatal("Could not open dataset: ", err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	buf := make([]byte, 0, 64*1024)
	scanner.Buffer(buf, 10*1024*1024)

	count := 0
	fmt.Println("Starting the massive 5.3GB import (Optimized Batch Mode)...")

	// Begin the first transaction
	tx, err := db.Begin()
	if err != nil {
		log.Fatal(err)
	}

	for scanner.Scan() {
		line := scanner.Bytes()
		var review YelpReviewJSON
		if err := json.Unmarshal(line, &review); err != nil {
			continue
		}

		_, err = tx.Exec(`
			INSERT INTO yelp_reviews (review_id, user_id, business_id, stars, review_date, review_text)
			VALUES ($1, $2, $3, $4, $5, $6)
			ON CONFLICT (review_id) DO NOTHING;
		`, review.ReviewID, review.UserID, review.BusinessID, int(review.Stars), review.Date, review.Text)

		if err != nil {
			log.Println("Failed to insert review:", err)
		}

		count++

		if count%500 == 0 {
			fmt.Printf("Inserted %d reviews...\n", count)
		}

		if count >= 2000 {
			break
		}
	}

	// Commit any remaining reviews
	if err := tx.Commit(); err != nil {
		log.Println("Final commit failed:", err)
	}

	if err := scanner.Err(); err != nil {
		log.Fatal("Error reading file:", err)
	}

	fmt.Println("🎉 Seeding Complete! Total:", count)
}
