package performance_analytics

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"math/rand"
	"net/http"
	"os"
	"time"

	"github.com/joho/godotenv"
)

type InventoryResponse struct {
	Elements []Item `json:"elements"`
}

type TendersResponse struct {
	Elements []struct {
		ID    string `json:"id"`
		Label string `json:"label"`
	} `json:"elements"`
}

type Item struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Price int    `json:"price"`
}

type OrderResponse struct {
	ID string `json:"id"`
}

func SeedDailyCloverData() {
	// Load environment variables from .env
	if err := godotenv.Overload(); err != nil {
		fmt.Printf("Warning: Error loading .env file: %v\n", err)
	}

	merchantID := os.Getenv("CLOVER_MERCHANT_MID")
	apiToken := os.Getenv("CLOVER_DEV_API_KEY")
	baseURL := "https://apisandbox.dev.clover.com/v3/merchants/"

	fmt.Printf("DEBUG: merchantID = '%s', apiToken = '%s'\n", merchantID, apiToken)

	if merchantID == "" || apiToken == "" {
		fmt.Println("Error: CLOVER_MERCHANT_MID and CLOVER_DEV_API_KEY must be set in your .env file.")
		return
	}

	// Initialize the random seed
	rand.Seed(time.Now().UnixNano())
	client := &http.Client{}

	// 1. Fetch real inventory IDs from Clover
	req, _ := http.NewRequest("GET", baseURL+merchantID+"/items", nil)
	req.Header.Add("Authorization", "Bearer "+apiToken)
	req.Header.Add("accept", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		fmt.Println("Error fetching inventory:", err)
		return
	}
	defer resp.Body.Close()

	body, _ := ioutil.ReadAll(resp.Body)
	var inv InventoryResponse
	json.Unmarshal(body, &inv)

	if len(inv.Elements) == 0 {
		fmt.Println("No items found. Ensure the spreadsheet was imported.")
		return
	}
	fmt.Printf("Loaded %d items from inventory for seeding.\n", len(inv.Elements))

	// 1.5 Fetch the ID for the "Cash" tender
	reqTender, _ := http.NewRequest("GET", baseURL+merchantID+"/tenders", nil)
	reqTender.Header.Add("Authorization", "Bearer "+apiToken)
	reqTender.Header.Add("accept", "application/json")

	respTender, errTender := client.Do(reqTender)
	if errTender != nil {
		fmt.Println("Error fetching tenders:", errTender)
		return
	}
	defer respTender.Body.Close()
	bodyTender, _ := ioutil.ReadAll(respTender.Body)

	var tenders TendersResponse
	json.Unmarshal(bodyTender, &tenders)

	var cashTenderID string
	for _, t := range tenders.Elements {
		if t.Label == "Cash" {
			cashTenderID = t.ID
			break
		}
	}

	if cashTenderID == "" {
		fmt.Println("Error: Could not find a Cash tender.")
		return
	}
	fmt.Println("Found Cash Tender ID:", cashTenderID)

	// 2. Loop 10 times to generate a realistic batch of DAILY orders
	for i := 1; i <= 10; i++ {
		// Pass 0 to generate times for TODAY
		historicalTime := generateRealisticSalesTime(0)

		// Pass the backdated timestamp in the payload
		orderPayload := fmt.Sprintf(`{"clientCreatedTime": %d, "state": "locked"}`, historicalTime)

		orderReq, _ := http.NewRequest("POST", baseURL+merchantID+"/orders", bytes.NewBuffer([]byte(orderPayload)))
		orderReq.Header.Add("Authorization", "Bearer "+apiToken)
		orderReq.Header.Add("accept", "application/json")
		orderReq.Header.Add("content-type", "application/json")

		orderResp, _ := client.Do(orderReq)
		orderBody, _ := ioutil.ReadAll(orderResp.Body)
		orderResp.Body.Close()

		var newOrder OrderResponse
		json.Unmarshal(orderBody, &newOrder)

		// 3. Pick 1 to 4 random items to simulate a real ticket
		numItems := rand.Intn(4) + 1
		orderTotal := 0
		for j := 0; j < numItems; j++ {
			randomItem := inv.Elements[rand.Intn(len(inv.Elements))]
			orderTotal += randomItem.Price

			lineItemPayload := fmt.Sprintf(`{"item": {"id": "%s"}}`, randomItem.ID)
			lineReq, _ := http.NewRequest("POST", baseURL+merchantID+"/orders/"+newOrder.ID+"/line_items", bytes.NewBuffer([]byte(lineItemPayload)))
			lineReq.Header.Add("Authorization", "Bearer "+apiToken)
			lineReq.Header.Add("accept", "application/json")
			lineReq.Header.Add("content-type", "application/json")

			client.Do(lineReq)
		}

		// Update the order's total in Clover
		updatePayload := fmt.Sprintf(`{"total": %d}`, orderTotal)
		updateReq, _ := http.NewRequest("POST", baseURL+merchantID+"/orders/"+newOrder.ID, bytes.NewBuffer([]byte(updatePayload)))
		updateReq.Header.Add("Authorization", "Bearer "+apiToken)
		updateReq.Header.Add("content-type", "application/json")
		client.Do(updateReq)

		// Pay the order immediately
		paymentPayload := fmt.Sprintf(`{"amount": %d, "tender": {"id": "%s"}}`, orderTotal, cashTenderID)
		payReq, _ := http.NewRequest("POST", baseURL+merchantID+"/orders/"+newOrder.ID+"/payments", bytes.NewBuffer([]byte(paymentPayload)))
		payReq.Header.Add("Authorization", "Bearer "+apiToken)
		payReq.Header.Add("accept", "application/json")
		payReq.Header.Add("content-type", "application/json")
		client.Do(payReq)

		fmt.Printf("Created & Paid Mock Order #%d (ID: %s) with %d items. Total: %d cents. (Mocked Time: %d)\n", i, newOrder.ID, numItems, orderTotal, historicalTime)

		// Sleep to avoid hitting Clover's API rate limits
		time.Sleep(300 * time.Millisecond)
	}
	fmt.Println("Successfully seeded today's mock orders!")
}

// generateRealisticSalesTime calculates a timestamp weighted toward restaurant rush hours
func generateRealisticSalesTime(daysBack int) int64 {
	now := time.Now()
	randomDaysBack := rand.Intn(daysBack + 1)
	targetDate := now.AddDate(0, 0, -randomDaysBack)

	var hour int
	// 50% chance for lunch rush, 50% chance for dinner rush
	if rand.Intn(2) == 0 {
		hour = rand.Intn(4) + 11 // 11 AM to 2 PM
	} else {
		hour = rand.Intn(5) + 17 // 5 PM to 9 PM
	}

	minute := rand.Intn(60)
	second := rand.Intn(60)

	finalTime := time.Date(targetDate.Year(), targetDate.Month(), targetDate.Day(), hour, minute, second, 0, targetDate.Location())

	// Clover requires time in Unix milliseconds
	return finalTime.UnixNano() / int64(time.Millisecond)
}
