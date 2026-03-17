//go:build ignore

// Example: Anthropic computer use tools.
//
// Anthropic provides built-in tools for computer interaction:
// Computer (mouse/keyboard), Bash (shell commands), TextEditor (file editing).
// These are provider-defined tools -- Anthropic handles the schema, you handle execution.
//
// Via Anthropic direct:
//
//	export ANTHROPIC_API_KEY=...
//	go run examples/computer-use/main.go anthropic
//
// Via AWS Bedrock:
//
//	export AWS_ACCESS_KEY_ID=... AWS_SECRET_ACCESS_KEY=... AWS_REGION=...
//	go run examples/computer-use/main.go bedrock
package main

import (
	"context"
	"encoding/json"
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

	// Define computer use tools with screen dimensions.
	// anthropic.Tools works with both the Anthropic and Bedrock providers.
	computerDef := anthropic.Tools.Computer(anthropic.ComputerToolOptions{
		DisplayWidthPx:  1920,
		DisplayHeightPx: 1080,
	})
	bashDef := anthropic.Tools.Bash()
	textEditorDef := anthropic.Tools.TextEditor()

	// Convert provider-defined tools to goai.Tool with Execute handlers.
	// In production, Execute would capture screenshots, run commands, etc.
	computerTool := goai.Tool{
		Name:                   computerDef.Name,
		ProviderDefinedType:    computerDef.ProviderDefinedType,
		ProviderDefinedOptions: computerDef.ProviderDefinedOptions,
		Execute: func(ctx context.Context, input json.RawMessage) (string, error) {
			fmt.Printf("  [computer] action: %s\n", string(input))
			return "Action executed successfully.", nil
		},
	}

	bashTool := goai.Tool{
		Name:                bashDef.Name,
		ProviderDefinedType: bashDef.ProviderDefinedType,
		Execute: func(ctx context.Context, input json.RawMessage) (string, error) {
			var args struct {
				Command string `json:"command"`
			}
			json.Unmarshal(input, &args)
			fmt.Printf("  [bash] command: %s\n", args.Command)
			return "$ " + args.Command + "\ncommand output here", nil
		},
	}

	textEditorTool := goai.Tool{
		Name:                textEditorDef.Name,
		ProviderDefinedType: textEditorDef.ProviderDefinedType,
		Execute: func(ctx context.Context, input json.RawMessage) (string, error) {
			fmt.Printf("  [editor] action: %s\n", string(input))
			return "File operation completed.", nil
		},
	}

	fmt.Printf("Using %s provider\n", auth)
	fmt.Println("Asking Claude to list files using bash tool...")
	fmt.Println()

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	result, err := goai.GenerateText(ctx, model,
		goai.WithPrompt("List the files in the current directory using the bash tool."),
		goai.WithTools(computerTool, bashTool, textEditorTool),
		goai.WithMaxSteps(3),
	)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println()
	fmt.Println("Result:", result.Text)
	fmt.Printf("Steps: %d, Tokens: %d in, %d out\n",
		len(result.Steps), result.TotalUsage.InputTokens, result.TotalUsage.OutputTokens)
}
