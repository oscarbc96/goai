//go:build ignore

// Example: Anthropic Code Execution tool.
//
// Claude writes and runs Python code in a sandboxed environment.
// Via Anthropic direct API, code runs server-side.
// Via Bedrock, the tool is accepted but runs as a function call (client-side execution needed).
//
// Via Anthropic direct:
//
//	export ANTHROPIC_API_KEY=...
//	go run examples/code-execution/main.go anthropic
//
// Via AWS Bedrock:
//
//	export AWS_ACCESS_KEY_ID=... AWS_SECRET_ACCESS_KEY=... AWS_REGION=...
//	go run examples/code-execution/main.go bedrock
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

	// code_execution_20260120 -- GA, no beta header needed.
	def := anthropic.Tools.CodeExecution()

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	result, err := goai.GenerateText(ctx, model,
		goai.WithPrompt("Calculate the sum of all prime numbers below 1000 using Python."),
		goai.WithMaxOutputTokens(2000),
		goai.WithTools(goai.Tool{
			Name:                def.Name,
			ProviderDefinedType: def.ProviderDefinedType,
		}),
		goai.WithMaxSteps(3),
	)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println(result.Text)
	fmt.Printf("\nSteps: %d, Usage: %d in, %d out\n",
		len(result.Steps), result.TotalUsage.InputTokens, result.TotalUsage.OutputTokens)
}
