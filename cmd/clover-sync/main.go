package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"math/rand"
	"net/http"
	"os"
	"strings"
	"time"

	"fbperformance/internal/cache"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
)

type CloverItem struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Price int    `json:"price"`
}

type CloverItemsResponse struct {
	Elements []CloverItem `json:"elements"`
}

type CloverOrder struct {
	ID           string `json:"id"`
	Total        int    `json:"total"`
	CreatedTime  int64  `json:"createdTime"`
	PaymentState string `json:"paymentState"`
	State        string `json:"state"`
	OrderType    struct {
		Name string `json:"name"`
	} `json:"orderType"`
	LineItems struct {
		Elements []struct {
			Item struct {
				ID string `json:"id"`
			} `json:"item"`
			Price int `json:"price"`
		} `json:"elements"`
	} `json:"lineItems"`
}

type CloverOrdersResponse struct {
	Elements []CloverOrder `json:"elements"`
}

func categorizeItem(name string) string {
	lower := strings.ToLower(name)
	if strings.Contains(lower, "cake") || strings.Contains(lower, "flan") || strings.Contains(lower, "donut") || strings.Contains(lower, "ice cream") {
		return "Desserts"
	}
	if strings.Contains(lower, "tea") || strings.Contains(lower, "coffee") || strings.Contains(lower, "juice") || strings.Contains(lower, "soda") || strings.Contains(lower, "water") {
		return "Beverages"
	}
	if strings.Contains(lower, "broth") || strings.Contains(lower, "salad") || strings.Contains(lower, "bok choy") || strings.Contains(lower, "brocollini") || strings.Contains(lower, "rice") || strings.Contains(lower, "noodle") || strings.Contains(lower, "egg") || strings.Contains(lower, "avocado") || strings.Contains(lower, "baguette") || strings.Contains(lower, "veggie") {
		return "Sides & Soups"
	}
	if strings.Contains(lower, "pot") || strings.Contains(lower, "ribs") || strings.Contains(lower, "mignon") || strings.Contains(lower, "crepe") || strings.Contains(lower, "duck") || strings.Contains(lower, "beef") || strings.Contains(lower, "chicken") || strings.Contains(lower, "pork") {
		return "Main Course"
	}
	
	// Fallback to a random category to ensure distribution
	cats := []string{"Appetizers", "Main Course", "Chef Specials"}
	return cats[rand.Intn(len(cats))]
}

func main() {
	godotenv.Load()
	cache.InitializeRedis()
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		dbURL = "postgres://postgres:devpassword@postgres:5432/fbperformance?sslmode=disable"
	}

	merchantID := os.Getenv("CLOVER_MERCHANT_MID")
	apiToken := os.Getenv("CLOVER_DEV_API_KEY")
	baseURL := "https://apisandbox.dev.clover.com/v3/merchants/"

	if merchantID == "" || apiToken == "" {
		log.Fatal("CLOVER_MERCHANT_MID and CLOVER_DEV_API_KEY must be set in .env")
	}

	pool, err := pgxpool.New(context.Background(), dbURL)
	if err != nil {
		log.Fatalf("failed to connect to db: %v", err)
	}
	defer pool.Close()

	ctx := context.Background()

	client := &http.Client{Timeout: 30 * time.Second}

	// 1. Sync Items
	log.Println("Syncing inventory from Clover...")
	req, _ := http.NewRequest("GET", baseURL+merchantID+"/items?limit=1000", nil)
	req.Header.Add("Authorization", "Bearer "+apiToken)
	req.Header.Add("accept", "application/json")
	resp, err := client.Do(req)
	if err != nil {
		log.Fatalf("failed to fetch items: %v", err)
	}
	defer resp.Body.Close()
	body, _ := ioutil.ReadAll(resp.Body)

	var itemsResp CloverItemsResponse
	json.Unmarshal(body, &itemsResp)

	for _, item := range itemsResp.Elements {
		itemUUID := uuid.NewMD5(uuid.NameSpaceOID, []byte(item.ID))
		price := float64(item.Price) / 100.0
		// Sandbox doesn't have categories, so we infer them from the name
		category := categorizeItem(item.Name)
		cogs := price * 0.25

		_, err := pool.Exec(ctx, `
			INSERT INTO menu_items (id, name, category, current_price, cogs)
			VALUES ($1, $2, $3, $4, $5)
			ON CONFLICT (id) DO UPDATE SET
				name = EXCLUDED.name,
				current_price = EXCLUDED.current_price
		`, itemUUID, item.Name, category, price, cogs)
		if err != nil {
			log.Printf("Failed to upsert item %s: %v", item.Name, err)
		}
	}
	log.Printf("Successfully synced %d items.\n", len(itemsResp.Elements))

	// 2. Sync Transactions
	log.Println("Syncing orders from Clover...")
	// Pull last 90 days of orders to have enough trend history
	fromMs := time.Now().AddDate(0, 0, -90).UnixMilli()
	toMs := time.Now().AddDate(0, 0, 1).UnixMilli()

	ordersURL := fmt.Sprintf("%s%s/orders?filter=createdTime>=%d&filter=createdTime<=%d&expand=lineItems&limit=1000", baseURL, merchantID, fromMs, toMs)
	reqOrders, _ := http.NewRequest("GET", ordersURL, nil)
	reqOrders.Header.Add("Authorization", "Bearer "+apiToken)
	reqOrders.Header.Add("accept", "application/json")
	respOrders, err := client.Do(reqOrders)
	if err != nil {
		log.Fatalf("failed to fetch orders: %v", err)
	}
	defer respOrders.Body.Close()
	bodyOrders, _ := ioutil.ReadAll(respOrders.Body)

	var ordersResp CloverOrdersResponse
	json.Unmarshal(bodyOrders, &ordersResp)

	// Clear out old transactions
	_, err = pool.Exec(ctx, "TRUNCATE TABLE transactions")
	if err != nil {
		log.Fatalf("Failed to truncate transactions: %v", err)
	}

	txCount := 0
	for _, order := range ordersResp.Elements {
		if order.PaymentState != "PAID" && order.PaymentState != "OPEN" && order.State != "locked" {
			continue
		}

		soldAt := time.UnixMilli(order.CreatedTime)
		
		for _, lineItem := range order.LineItems.Elements {
			if lineItem.Item.ID == "" {
				continue
			}
			itemUUID := uuid.NewMD5(uuid.NameSpaceOID, []byte(lineItem.Item.ID))
			price := float64(lineItem.Price) / 100.0

			_, err := pool.Exec(ctx, `
				INSERT INTO transactions (menu_item_id, quantity, unit_price, sold_at, order_type)
				VALUES ($1, $2, $3, $4, $5)
			`, itemUUID, 1, price, soldAt, order.OrderType.Name)
			if err != nil {
				// Don't log missing menu item errors since some items might have been deleted from inventory
			} else {
				txCount++
			}
		}
	}

	log.Printf("Successfully synced %d transactions from %d orders.\n", txCount, len(ordersResp.Elements))
	cache.InvalidateAnalyticsCache(ctx)
}
