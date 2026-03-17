//go:build ignore

// Example: tool definition with single-step tool call.
//
// Usage:
//
//	export GEMINI_API_KEY=...
//	go run examples/tools/main.go
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"

	"github.com/zendev-sh/goai"
	"github.com/zendev-sh/goai/provider/google"
)

func main() {
	model := google.Chat("gemini-2.0-flash", google.WithAPIKey(os.Getenv("GEMINI_API_KEY")))

	weatherTool := goai.Tool{
		Name:        "get_weather",
		Description: "Get the current weather for a city.",
		InputSchema: json.RawMessage(`{
			"type": "object",
			"properties": {
				"city": {"type": "string", "description": "City name"}
			},
			"required": ["city"]
		}`),
		Execute: func(ctx context.Context, input json.RawMessage) (string, error) {
			var args struct {
				City string `json:"city"`
			}
			if err := json.Unmarshal(input, &args); err != nil {
				return "", err
			}
			// Simulated weather lookup.
			return fmt.Sprintf("The weather in %s is 22°C and sunny.", args.City), nil
		},
	}

	// MaxSteps=2: step 1 = model calls tool, step 2 = model uses result.
	result, err := goai.GenerateText(context.Background(), model,
		goai.WithPrompt("What's the weather like in Tokyo?"),
		goai.WithTools(weatherTool),
		goai.WithMaxSteps(2),
	)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println(result.Text)
	fmt.Printf("Steps: %d, Tokens: %d in, %d out\n",
		len(result.Steps), result.TotalUsage.InputTokens, result.TotalUsage.OutputTokens)
}
