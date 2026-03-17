package goai

import (
	"testing"

	"github.com/zendev-sh/goai/provider"
)

func TestSystemMessage(t *testing.T) {
	msg := SystemMessage("You are helpful.")
	if msg.Role != provider.RoleSystem {
		t.Errorf("Role = %v, want system", msg.Role)
	}
	if len(msg.Content) != 1 {
		t.Fatalf("Content = %d parts, want 1", len(msg.Content))
	}
	if msg.Content[0].Type != provider.PartText {
		t.Errorf("Type = %v, want text", msg.Content[0].Type)
	}
	if msg.Content[0].Text != "You are helpful." {
		t.Errorf("Text = %q", msg.Content[0].Text)
	}
}

func TestUserMessage(t *testing.T) {
	msg := UserMessage("hello")
	if msg.Role != provider.RoleUser {
		t.Errorf("Role = %v, want user", msg.Role)
	}
	if msg.Content[0].Text != "hello" {
		t.Errorf("Text = %q", msg.Content[0].Text)
	}
}

func TestAssistantMessage(t *testing.T) {
	msg := AssistantMessage("I can help.")
	if msg.Role != provider.RoleAssistant {
		t.Errorf("Role = %v, want assistant", msg.Role)
	}
	if msg.Content[0].Text != "I can help." {
		t.Errorf("Text = %q", msg.Content[0].Text)
	}
}

func TestToolMessage(t *testing.T) {
	msg := ToolMessage("call_1", "read_file", "file contents here")
	if msg.Role != provider.RoleTool {
		t.Errorf("Role = %v, want tool", msg.Role)
	}
	if len(msg.Content) != 1 {
		t.Fatalf("Content = %d parts, want 1", len(msg.Content))
	}
	part := msg.Content[0]
	if part.Type != provider.PartToolResult {
		t.Errorf("Type = %v, want tool-result", part.Type)
	}
	if part.ToolCallID != "call_1" {
		t.Errorf("ToolCallID = %q", part.ToolCallID)
	}
	if part.ToolName != "read_file" {
		t.Errorf("ToolName = %q", part.ToolName)
	}
	if part.ToolOutput != "file contents here" {
		t.Errorf("ToolOutput = %q", part.ToolOutput)
	}
}
