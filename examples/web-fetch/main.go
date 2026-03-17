//go:build ignore

// Example: Anthropic Web Fetch tool.
//
// Claude fetches and processes web content from specific URLs.
// Different from web search -- this fetches a known URL, not search results.
// Via Anthropic direct API, fetching runs server-side.
// Via Bedrock, the tool is accepted but the model generates a tool_call for client-side handling.
//
// Via Anthropic direct:
//
//	export ANTHROPIC_API_KEY=...
//	go run examples/web-fetch/main.go anthropic
//
// Via AWS Bedrock:
//
//	export AWS_ACCESS_KEY_ID=... AWS_SECRET_ACCESS_KEY=... AWS_REGION=...
//	go run examples/web-fetch/main.go bedrock
package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/zendev-sh/goai"
	"github.com/zendev-sh/goai/provider"
	"github.com/zendev-sh/goai/provider/anthropic"
	"github.com/zendev-sh/goai/provider/bedrock"
)

func main() {
	auth := "bedrock"
	if len(os.Args) > 1 {
		auth = os.Args[1]
	}

	var model provider.LanguageModel
	switch auth {
	case "anthropic":
		model = anthropic.Chat("claude-sonnet-4-20250514",
			anthropic.WithAPIKey(os.Getenv("ANTHROPIC_API_KEY")))
	case "bedrock":
		model = bedrock.Chat("anthropic.claude-sonnet-4-20250514-v1:0",
			bedrock.WithAccessKey(os.Getenv("AWS_ACCESS_KEY_ID")),
			bedrock.WithSecretKey(os.Getenv("AWS_SECRET_ACCESS_KEY")),
			bedrock.WithRegion(os.Getenv("AWS_REGION")))
	default:
		log.Fatalf("Unknown: %s (use anthropic or bedrock)", auth)
	}

	def := anthropic.Tools.WebFetch(
		anthropic.WithWebFetchMaxUses(3),
		anthropic.WithCitations(true),
	)

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	result, err := goai.GenerateText(ctx, model,
		goai.WithPrompt("Fetch https://go.dev/doc/go1.22 and summarize the key changes."),
		goai.WithMaxOutputTokens(1000),
		goai.WithTools(goai.Tool{
			Name:                   def.Name,
			ProviderDefinedType:    def.ProviderDefinedType,
			ProviderDefinedOptions: def.ProviderDefinedOptions,
		}),
		goai.WithMaxSteps(3),
	)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println(result.Text)
	fmt.Printf("\nSources: %d\n", len(result.Sources))
	for i, s := range result.Sources {
		fmt.Printf("  [%d] %s - %s\n", i, s.Title, s.URL)
	}
}
