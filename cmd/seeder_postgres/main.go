package main

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"os"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
)

var categories = []string{"Appetizers", "Main Course", "Desserts", "Beverages", "Specials"}
var items = []string{"Burger", "Pizza", "Salad", "Pasta", "Steak", "Fries", "Ice Cream", "Coke", "Water", "Wine"}

func main() {
	godotenv.Load()
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		dbURL = "postgres://postgres:devpassword@postgres:5432/fbperformance?sslmode=disable"
	}

	pool, err := pgxpool.New(context.Background(), dbURL)
	if err != nil {
		log.Fatalf("failed to connect: %v", err)
	}
	defer pool.Close()

	ctx := context.Background()

	// Clear existing data
	pool.Exec(ctx, "TRUNCATE TABLE transactions, menu_items CASCADE")

	var itemIDs []string
	for i := 0; i < 20; i++ {
		name := fmt.Sprintf("%s %s", items[rand.Intn(len(items))], items[rand.Intn(len(items))])
		category := categories[rand.Intn(len(categories))]
		price := float64(rand.Intn(40) + 10)
		cogs := price * (rand.Float64()*0.2 + 0.1) // 10-30% food cost

		var id string
		err := pool.QueryRow(ctx, "INSERT INTO menu_items (name, category, current_price, cogs) VALUES ($1, $2, $3, $4) RETURNING id",
			name, category, price, cogs).Scan(&id)
		if err != nil {
			log.Fatalf("insert item: %v", err)
		}
		itemIDs = append(itemIDs, id)
	}

	fmt.Printf("Seeded %d menu items\n", len(itemIDs))

	now := time.Now()
	txCount := 0
	for i := 0; i < 2000; i++ {
		itemID := itemIDs[rand.Intn(len(itemIDs))]
		qty := rand.Intn(5) + 1
		unitPrice := float64(rand.Intn(40) + 10)
		
		daysBack := rand.Intn(60)
		soldAt := now.AddDate(0, 0, -daysBack).Add(time.Duration(rand.Intn(24)) * time.Hour)

		_, err := pool.Exec(ctx, "INSERT INTO transactions (menu_item_id, quantity, unit_price, sold_at) VALUES ($1, $2, $3, $4)",
			itemID, qty, unitPrice, soldAt)
		if err != nil {
			log.Fatalf("insert tx: %v", err)
		}
		txCount++
	}
	fmt.Printf("Seeded %d transactions\n", txCount)
}
