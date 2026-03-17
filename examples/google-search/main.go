//go:build ignore

// Example: Google Search grounding with Gemini.
//
// Gemini searches Google and returns grounded responses with source citations.
// Sources are returned in result.Sources as URLs from Google Search.
//
// Usage:
//
//	export GEMINI_API_KEY=...
//	go run examples/google-search/main.go
package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/zendev-sh/goai"
	"github.com/zendev-sh/goai/provider/google"
)

func main() {
	model := google.Chat("gemini-2.5-flash",
		google.WithAPIKey(os.Getenv("GEMINI_API_KEY")))

	def := google.Tools.GoogleSearch()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	result, err := goai.GenerateText(ctx, model,
		goai.WithPrompt("What are the latest features in Go 1.26?"),
		goai.WithMaxOutputTokens(500),
		goai.WithTools(goai.Tool{
			Name:                   def.Name,
			ProviderDefinedType:    def.ProviderDefinedType,
			ProviderDefinedOptions: def.ProviderDefinedOptions,
		}),
	)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println(result.Text)
	fmt.Printf("\nGrounding sources: %d\n", len(result.Sources))
	for i, s := range result.Sources {
		fmt.Printf("  [%d] %s - %s\n", i, s.Title, s.URL)
	}
}
