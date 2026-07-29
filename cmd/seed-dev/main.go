// Command seed-dev inserts a small, realistic menu plus several weeks of
// transaction history into the local database, so the multi-agent
// recommendation pipeline (POST /api/ai/recommendation) and the standalone
// forecast endpoint (POST /api/forecast) can be exercised without
// hand-written INSERTs. It is safe to re-run: it skips creating a menu
// item that already exists by name, and independently skips seeding
// transactions for any item that already has some.
package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"math/rand"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/joho/godotenv"

	"fbperformance/internal/config"
)

// historyDays covers more than the forecast service's 30-day lookback
// window so every seeded item has a full history to forecast against.
const historyDays = 45

// seedItem describes one menu item and the daily-demand curve used to
// generate its transaction history. baseline is the average units sold on
// a weekday; trendPerDay is added per day of elapsed history, so a
// positive value ramps demand up toward today (biasing the Trend/Financial
// agents toward LAUNCH) and a negative value ramps it down (biasing
// toward CUT).
type seedItem struct {
	name        string
	category    string
	priceCents  int64
	cogsCents   int64
	baseline    float64
	trendPerDay float64
}

var seedItems = []seedItem{
	{name: "Bulgogi Bowl", category: "Entree", priceCents: 1499, cogsCents: 500, baseline: 12, trendPerDay: 0.3},
	{name: "Kimchi Fried Rice", category: "Entree", priceCents: 1250, cogsCents: 400, baseline: 10, trendPerDay: 0.1},
	{name: "Bibimbap", category: "Entree", priceCents: 1399, cogsCents: 475, baseline: 11, trendPerDay: 0.2},
	{name: "Jeyuk Bokkeum", category: "Entree", priceCents: 1350, cogsCents: 460, baseline: 9, trendPerDay: 0.15},
	{name: "Yangnyeom Fried Chicken", category: "Entree", priceCents: 1599, cogsCents: 550, baseline: 13, trendPerDay: 0.4},
	{name: "Japchae", category: "Entree", priceCents: 1199, cogsCents: 380, baseline: 7, trendPerDay: -0.05},
	{name: "Kimchijeon", category: "Appetizer", priceCents: 950, cogsCents: 300, baseline: 8, trendPerDay: 0},
	{name: "Mandu Dumplings", category: "Appetizer", priceCents: 899, cogsCents: 290, baseline: 10, trendPerDay: 0.1},
	{name: "Tteokbokki", category: "Appetizer", priceCents: 999, cogsCents: 320, baseline: 9, trendPerDay: 0.25},
	{name: "Korean Corn Dog", category: "Appetizer", priceCents: 750, cogsCents: 250, baseline: 14, trendPerDay: 0.5},
	{name: "Yuja Citron Tea", category: "Beverage", priceCents: 550, cogsCents: 180, baseline: 6, trendPerDay: 0.1},
	{name: "Sikhye", category: "Beverage", priceCents: 450, cogsCents: 120, baseline: 5, trendPerDay: 0},
	{name: "Injeolmi Bingsu", category: "Dessert", priceCents: 950, cogsCents: 320, baseline: 4, trendPerDay: -0.1},
}

func main() {
	_ = godotenv.Load()
	cfg := config.Load()

	databaseURL := cfg.DatabaseURL
	if databaseURL == "" {
		databaseURL = "postgres://postgres:devpassword@localhost:5440/fbperformance?sslmode=disable"
	}

	db, err := sql.Open("pgx", databaseURL)
	if err != nil {
		log.Fatalf("open database: %v", err)
	}
	defer db.Close()
	if err := db.Ping(); err != nil {
		log.Fatalf("ping database: %v", err)
	}

	rand.Seed(time.Now().UnixNano())

	ctx := context.Background()
	for _, item := range seedItems {
		id, err := ensureMenuItem(ctx, db, item)
		if err != nil {
			log.Fatalf("seed menu item %q: %v", item.name, err)
		}

		hasHistory, err := hasTransactionHistory(ctx, db, id)
		if err != nil {
			log.Fatalf("check transaction history for %q: %v", item.name, err)
		}
		if hasHistory {
			fmt.Printf("Skipping %-28s already has transaction history (id=%s)\n", item.name, id)
			continue
		}

		if err := seedTransactionHistory(ctx, db, id, item); err != nil {
			log.Fatalf("seed transactions for %q: %v", item.name, err)
		}
		fmt.Printf("Seeded  %-28s %d days of history (id=%s)\n", item.name, historyDays, id)
	}
	fmt.Println("\nDone. POST one of the ids above as menu_item_id to /api/ai/recommendation or /api/forecast.")
}

// ensureMenuItem inserts item unless a row with the same name already
// exists, so the seeder is safe to re-run. It returns the row's id either
// way — callers must check hasTransactionHistory separately, since a
// pre-existing row (from a prior interrupted run, or a manual INSERT)
// might not have any transactions yet.
func ensureMenuItem(ctx context.Context, db *sql.DB, item seedItem) (string, error) {
	var id string
	err := db.QueryRowContext(ctx, `SELECT id FROM menu_items WHERE name = $1`, item.name).Scan(&id)
	if err == nil {
		return id, nil
	}
	if err != sql.ErrNoRows {
		return "", err
	}

	err = db.QueryRowContext(ctx, `
		INSERT INTO menu_items (name, category, current_price, cogs, is_active)
		VALUES ($1, $2, $3::numeric / 100, $4::numeric / 100, true)
		RETURNING id`,
		item.name, item.category, item.priceCents, item.cogsCents,
	).Scan(&id)
	if err != nil {
		return "", err
	}
	return id, nil
}

// hasTransactionHistory reports whether the menu item already has at
// least one transactions row.
func hasTransactionHistory(ctx context.Context, db *sql.DB, menuItemID string) (bool, error) {
	var exists bool
	err := db.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM transactions WHERE menu_item_id = $1)`, menuItemID).Scan(&exists)
	return exists, err
}

// seedTransactionHistory writes one transaction row per day for the last
// historyDays days, with quantity driven by the item's baseline, linear
// trend, weekend upweighting, and random noise.
func seedTransactionHistory(ctx context.Context, db *sql.DB, menuItemID string, item seedItem) error {
	now := time.Now().UTC()
	for daysAgo := historyDays; daysAgo >= 0; daysAgo-- {
		day := now.AddDate(0, 0, -daysAgo)

		daysIntoHistory := float64(historyDays - daysAgo)
		expected := item.baseline + item.trendPerDay*daysIntoHistory
		if day.Weekday() == time.Friday || day.Weekday() == time.Saturday {
			expected *= 1.4
		}
		quantity := int(expected + rand.NormFloat64()*(expected*0.2))
		if quantity <= 0 {
			continue
		}

		soldAt := randomRushHour(day)
		if _, err := db.ExecContext(ctx, `
			INSERT INTO transactions (menu_item_id, quantity, unit_price, sold_at)
			VALUES ($1, $2, $3::numeric / 100, $4)`,
			menuItemID, quantity, item.priceCents, soldAt,
		); err != nil {
			return err
		}
	}
	return nil
}

// randomRushHour returns a timestamp on the given day weighted toward
// lunch (11am-2pm) or dinner (5pm-9pm) service.
func randomRushHour(day time.Time) time.Time {
	var hour int
	if rand.Intn(2) == 0 {
		hour = 11 + rand.Intn(4)
	} else {
		hour = 17 + rand.Intn(5)
	}
	return time.Date(day.Year(), day.Month(), day.Day(), hour, rand.Intn(60), rand.Intn(60), 0, time.UTC)
}
