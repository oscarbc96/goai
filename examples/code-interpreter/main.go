//go:build ignore

// Example: OpenAI Code Interpreter tool.
//
// The model writes and executes Python code in a sandboxed container.
// Server-side execution -- no local setup needed.
//
// Via OpenAI direct:
//
//	export OPENAI_API_KEY=...
//	go run examples/code-interpreter/main.go openai
//
// Via Azure OpenAI:
//
//	export AZURE_OPENAI_API_KEY=... AZURE_RESOURCE_NAME=...
//	go run examples/code-interpreter/main.go azure
package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/zendev-sh/goai"
	"github.com/zendev-sh/goai/provider"
	"github.com/zendev-sh/goai/provider/azure"
	"github.com/zendev-sh/goai/provider/openai"
)

func main() {
	auth := "azure"
	if len(os.Args) > 1 {
		auth = os.Args[1]
	}

	var model provider.LanguageModel
	switch auth {
	case "openai":
		model = openai.Chat("gpt-4.1-mini",
			openai.WithAPIKey(os.Getenv("OPENAI_API_KEY")))
	case "azure":
		model = azure.Chat("gpt-4.1-mini",
			azure.WithAPIKey(os.Getenv("AZURE_OPENAI_API_KEY")))
	default:
		log.Fatalf("Unknown: %s (use openai or azure)", auth)
	}

	def := openai.Tools.CodeInterpreter()

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	result, err := goai.GenerateText(ctx, model,
		goai.WithPrompt("Calculate the first 20 Fibonacci numbers and show them as a formatted table."),
		goai.WithMaxOutputTokens(2000),
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
	fmt.Printf("\nUsage: %d in, %d out\n", result.TotalUsage.InputTokens, result.TotalUsage.OutputTokens)
}
