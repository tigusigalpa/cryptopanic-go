package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"time"

	cryptopanic "github.com/tigusigalpa/cryptopanic-go"
)

// PortfolioRetrieval demonstrates fetching the user's portfolio.
// This endpoint is available under Growth and Enterprise plans.
func main() {
	token := os.Getenv("CRYPTOPANIC_AUTH_TOKEN")
	if token == "" {
		log.Fatal("CRYPTOPANIC_AUTH_TOKEN is not set. Copy .env.example to .env and add your token.")
	}

	client := cryptopanic.New(token,
		cryptopanic.WithPlan(cryptopanic.PlanGrowth),
		cryptopanic.WithTimeout(10*time.Second),
	)

	resp, err := client.Portfolio(context.Background())
	if err != nil {
		log.Fatalf("failed to fetch portfolio: %v", err)
	}

	var pretty map[string]interface{}
	if err := json.Unmarshal(resp.Raw, &pretty); err != nil {
		log.Fatalf("failed to parse portfolio: %v", err)
	}

	encoded, _ := json.MarshalIndent(pretty, "", "  ")
	fmt.Println("Portfolio:")
	fmt.Println(string(encoded))
}
