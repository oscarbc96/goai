package goai

import (
	"testing"
	"time"

	"github.com/zendev-sh/goai/provider"
)

func TestDefaultOptions(t *testing.T) {
	o := defaultOptions()
	if o.MaxSteps != 1 {
		t.Errorf("MaxSteps = %d, want 1", o.MaxSteps)
	}
	if o.MaxRetries != 2 {
		t.Errorf("MaxRetries = %d, want 2", o.MaxRetries)
	}
}

func TestApplyOptions(t *testing.T) {
	o := applyOptions(
		WithSystem("You are helpful."),
		WithPrompt("hello"),
		WithMaxOutputTokens(1000),
		WithTemperature(0.7),
		WithTopP(0.9),
		WithStopSequences("END", "STOP"),
		WithMaxSteps(5),
		WithMaxRetries(3),
		WithTimeout(30*time.Second),
		WithPromptCaching(true),
		WithToolChoice("auto"),
		WithHeaders(map[string]string{"X-Custom": "value"}),
		WithProviderOptions(map[string]any{"key": "val"}),
	)

	if o.System != "You are helpful." {
		t.Errorf("System = %q", o.System)
	}
	if o.Prompt != "hello" {
		t.Errorf("Prompt = %q", o.Prompt)
	}
	if o.MaxOutputTokens != 1000 {
		t.Errorf("MaxOutputTokens = %d", o.MaxOutputTokens)
	}
	if o.Temperature == nil || *o.Temperature != 0.7 {
		t.Errorf("Temperature = %v", o.Temperature)
	}
	if o.TopP == nil || *o.TopP != 0.9 {
		t.Errorf("TopP = %v", o.TopP)
	}
	if len(o.StopSequences) != 2 {
		t.Errorf("StopSequences = %v", o.StopSequences)
	}
	if o.MaxSteps != 5 {
		t.Errorf("MaxSteps = %d", o.MaxSteps)
	}
	if o.MaxRetries != 3 {
		t.Errorf("MaxRetries = %d", o.MaxRetries)
	}
	if o.Timeout != 30*time.Second {
		t.Errorf("Timeout = %v", o.Timeout)
	}
	if !o.PromptCaching {
		t.Error("PromptCaching should be true")
	}
	if o.ToolChoice != "auto" {
		t.Errorf("ToolChoice = %q", o.ToolChoice)
	}
	if o.Headers["X-Custom"] != "value" {
		t.Errorf("Headers = %v", o.Headers)
	}
	if o.ProviderOptions["key"] != "val" {
		t.Errorf("ProviderOptions = %v", o.ProviderOptions)
	}
}

func TestWithMessages(t *testing.T) {
	msgs := []provider.Message{
		{Role: provider.RoleUser, Content: []provider.Part{{Type: provider.PartText, Text: "hi"}}},
		{Role: provider.RoleAssistant, Content: []provider.Part{{Type: provider.PartText, Text: "hello"}}},
	}

	o := applyOptions(WithMessages(msgs...))
	if len(o.Messages) != 2 {
		t.Fatalf("Messages = %d, want 2", len(o.Messages))
	}
	if o.Messages[0].Role != provider.RoleUser {
		t.Errorf("Messages[0].Role = %v", o.Messages[0].Role)
	}
}

func TestWithTools(t *testing.T) {
	tool := Tool{
		Name:        "read",
		Description: "Read a file",
	}
	o := applyOptions(WithTools(tool))
	if len(o.Tools) != 1 {
		t.Fatalf("Tools = %d, want 1", len(o.Tools))
	}
	if o.Tools[0].Name != "read" {
		t.Errorf("Tools[0].Name = %q", o.Tools[0].Name)
	}
}

func TestWithTemperature_Zero(t *testing.T) {
	o := applyOptions(WithTemperature(0.0))
	if o.Temperature == nil {
		t.Fatal("Temperature should not be nil")
	}
	if *o.Temperature != 0.0 {
		t.Errorf("Temperature = %v, want 0.0", *o.Temperature)
	}
}

func TestApplyOptions_Defaults(t *testing.T) {
	// No options applied -- should get defaults.
	o := applyOptions()
	if o.Temperature != nil {
		t.Errorf("Temperature should be nil by default, got %v", o.Temperature)
	}
	if o.TopP != nil {
		t.Errorf("TopP should be nil by default, got %v", o.TopP)
	}
	if o.System != "" {
		t.Errorf("System should be empty by default")
	}
	if len(o.Messages) != 0 {
		t.Errorf("Messages should be empty by default")
	}
}
