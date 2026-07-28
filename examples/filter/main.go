package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	cryptopanic "github.com/tigusigalpa/cryptopanic-go"
)

// FilterByCurrencies demonstrates filtering posts by specific
// currencies and using a filter.
func main() {
	token := os.Getenv("CRYPTOPANIC_AUTH_TOKEN")
	if token == "" {
		log.Fatal("CRYPTOPANIC_AUTH_TOKEN is not set. Copy .env.example to .env and add your token.")
	}

	client := cryptopanic.New(token,
		cryptopanic.WithPlan(cryptopanic.PlanGrowth),
		cryptopanic.WithTimeout(10*time.Second),
	)

	page, err := client.Posts(context.Background(), cryptopanic.PostsQuery{
		Currencies: []string{"BTC", "ETH"},
		Filter:     cryptopanic.FilterRising,
		Kind:       cryptopanic.KindNews,
	})
	if err != nil {
		log.Fatalf("failed to fetch posts: %v", err)
	}

	fmt.Printf("Rising BTC/ETH news (%d results):\n", len(page.Results))
	for _, post := range page.Results {
		fmt.Printf("  - %s (published: %s)\n", post.Title, post.PublishedAt)
	}
}
