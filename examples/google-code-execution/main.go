//go:build ignore

// Example: Google Gemini Code Execution tool.
//
// Gemini generates and runs Python code in a sandboxed environment.
// The code runs server-side on Google's infrastructure.
//
// Usage:
//
//	export GEMINI_API_KEY=...
//	go run examples/google-code-execution/main.go
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

	def := google.Tools.CodeExecution()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	result, err := goai.GenerateText(ctx, model,
		goai.WithPrompt("Write Python code to find the 50th Fibonacci number and verify it."),
		goai.WithMaxOutputTokens(1000),
		goai.WithTools(goai.Tool{
			Name:                def.Name,
			ProviderDefinedType: def.ProviderDefinedType,
		}),
	)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println(result.Text)
	fmt.Printf("\nUsage: %d in, %d out\n", result.TotalUsage.InputTokens, result.TotalUsage.OutputTokens)
}
