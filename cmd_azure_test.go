//go:build e2e

package goai_test

import (
	"context"
	"encoding/json"
	"os"
	"testing"
	"time"

	"github.com/zendev-sh/goai"
	"github.com/zendev-sh/goai/provider/azure"
)

func TestE2E_AzureClaude_Opus_ComputerUse(t *testing.T) {
	key := os.Getenv("AZURE_OPENAI_API_KEY")
	res := os.Getenv("AZURE_RESOURCE_NAME")
	if key == "" || res == "" {
		t.Skip("Azure credentials not set")
	}

	// azure.Chat() auto-detects Claude models and delegates to the
	// Anthropic endpoint at {resource}.services.ai.azure.com/anthropic.
	model := azure.Chat("claude-opus-4-6",
		azure.WithAPIKey(key))

	ctx, cancel := context.WithTimeout(t.Context(), 90*time.Second)
	defer cancel()

	toolInvoked := false

	result, err := goai.GenerateText(ctx, model,
		goai.WithPrompt("Use the text editor tool to view the file /tmp/test.txt. You MUST use the text_editor tool with command 'view'."),
		goai.WithMaxOutputTokens(300),
		goai.WithMaxSteps(3),
		goai.WithTools(goai.Tool{
			Name:                "str_replace_based_edit_tool",
			ProviderDefinedType: "text_editor_20250728",
			Execute: func(_ context.Context, input json.RawMessage) (string, error) {
				toolInvoked = true
				t.Logf("Text editor tool invoked: %s", string(input))
				return "File contents:\nHello from Azure Claude test!", nil
			},
		}),
	)
	if err != nil {
		t.Fatalf("error: %v", err)
	}

	t.Logf("Response.ID=%q Model=%q", result.Response.ID, result.Response.Model)
	t.Logf("Text=%q Steps=%d ToolInvoked=%v Sources=%d",
		result.Text, len(result.Steps), toolInvoked, len(result.Sources))

	if !toolInvoked {
		t.Error("expected text_editor tool to be invoked")
	}
	if result.Text == "" {
		t.Error("expected final text response")
	}
}
