package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	cryptopanic "github.com/tigusigalpa/cryptopanic-go"
)

// BasicNewsRetrieval demonstrates fetching the latest news posts
// from the CryptoPanic API.
func main() {
	token := os.Getenv("CRYPTOPANIC_AUTH_TOKEN")
	if token == "" {
		log.Fatal("CRYPTOPANIC_AUTH_TOKEN is not set. Copy .env.example to .env and add your token.")
	}

	client, err := cryptopanic.NewFromEnv(
		cryptopanic.WithPlan(cryptopanic.PlanGrowth),
		cryptopanic.WithTimeout(10*time.Second),
	)
	if err != nil {
		log.Fatalf("failed to create client: %v", err)
	}

	page, err := client.Posts(context.Background(), cryptopanic.PostsQuery{})
	if err != nil {
		log.Fatalf("failed to fetch posts: %v", err)
	}

	fmt.Printf("Retrieved %d posts\n", len(page.Results))
	for _, post := range page.Results {
		fmt.Printf("  [%s] %s\n", post.Kind, post.Title)
	}
}
