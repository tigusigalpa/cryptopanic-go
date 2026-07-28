package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	cryptopanic "github.com/tigusigalpa/cryptopanic-go"
)

// RSSRetrieval demonstrates fetching the CryptoPanic RSS feed.
// The reference says format=rss returns 20 items regardless of plan.
func main() {
	token := os.Getenv("CRYPTOPANIC_AUTH_TOKEN")
	if token == "" {
		log.Fatal("CRYPTOPANIC_AUTH_TOKEN is not set. Copy .env.example to .env and add your token.")
	}

	client := cryptopanic.New(token,
		cryptopanic.WithPlan(cryptopanic.PlanGrowth),
		cryptopanic.WithTimeout(10*time.Second),
	)

	// Method 1: Posts with format=rss
	fmt.Println("=== Posts RSS ===")
	rssResp, err := client.PostsRSS(context.Background(), cryptopanic.PostsQuery{
		Currencies: []string{"BTC"},
	})
	if err != nil {
		log.Fatalf("failed to fetch posts RSS: %v", err)
	}
	fmt.Printf("RSS content length: %d bytes\n", len(rssResp.Body))
	fmt.Println(rssResp.Body[:min(200, len(rssResp.Body))])
	fmt.Println("...")

	// Method 2: Public news RSS feed
	fmt.Println("\n=== News RSS ===")
	newsResp, err := client.NewsRSS(context.Background())
	if err != nil {
		log.Fatalf("failed to fetch news RSS: %v", err)
	}
	fmt.Printf("News RSS content length: %d bytes\n", len(newsResp.Body))
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
