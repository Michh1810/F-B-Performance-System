package main

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"

	"github.com/joho/godotenv"
)

func main() {
	godotenv.Load(".env")
	apiToken := os.Getenv("CLOVER_DEV_API_KEY")
	merchantID := os.Getenv("CLOVER_MERCHANT_MID")
	baseURL := "https://apisandbox.dev.clover.com/v3/merchants/"

	// Create order
	orderPayload := `{"state": "locked"}`
	orderReq, _ := http.NewRequest("POST", baseURL+merchantID+"/orders", bytes.NewBuffer([]byte(orderPayload)))
	orderReq.Header.Add("Authorization", "Bearer "+apiToken)
	orderReq.Header.Add("accept", "application/json")
	orderReq.Header.Add("content-type", "application/json")

	client := &http.Client{}
	orderResp, _ := client.Do(orderReq)
	orderBody, _ := ioutil.ReadAll(orderResp.Body)
	fmt.Println("Order Response:", string(orderBody))

	// Pretend order ID is extracted from orderBody...
	// Just for testing payment endpoint: Let's fetch an existing OPEN order
}
