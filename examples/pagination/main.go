package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	cryptopanic "github.com/tigusigalpa/cryptopanic-go"
)

// Pagination demonstrates fetching multiple pages of posts and
// using the pagination metadata.
func main() {
	token := os.Getenv("CRYPTOPANIC_AUTH_TOKEN")
	if token == "" {
		log.Fatal("CRYPTOPANIC_AUTH_TOKEN is not set. Copy .env.example to .env and add your token.")
	}

	client := cryptopanic.New(token,
		cryptopanic.WithPlan(cryptopanic.PlanGrowth),
		cryptopanic.WithTimeout(10*time.Second),
	)

	pageNum := 1
	for {
		page, err := client.Posts(context.Background(), cryptopanic.PostsQuery{
			Page: cryptopanic.IntPtr(pageNum),
		})
		if err != nil {
			log.Fatalf("failed to fetch page %d: %v", pageNum, err)
		}

		fmt.Printf("Page %d: %d posts\n", pageNum, len(page.Results))
		for _, post := range page.Results {
			fmt.Printf("  - %s\n", post.Title)
		}

		if !page.HasNext {
			fmt.Println("No more pages.")
			break
		}

		if page.NextPage != nil {
			pageNum = *page.NextPage
		} else {
			pageNum++
		}

		if pageNum > 3 {
			fmt.Println("Stopping after 3 pages for demo.")
			break
		}

		time.Sleep(500 * time.Millisecond)
	}
}
