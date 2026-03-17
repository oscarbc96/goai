package provider_test

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/zendev-sh/goai/provider"
)

// mockLanguageModel implements provider.LanguageModel for testing.
type mockLanguageModel struct {
	id   string
	caps provider.ModelCapabilities
}

func (m *mockLanguageModel) ModelID() string { return m.id }

func (m *mockLanguageModel) DoGenerate(_ context.Context, _ provider.GenerateParams) (*provider.GenerateResult, error) {
	return &provider.GenerateResult{
		Text:         "hello",
		FinishReason: provider.FinishStop,
		Usage:        provider.Usage{InputTokens: 10, OutputTokens: 5},
	}, nil
}

func (m *mockLanguageModel) DoStream(_ context.Context, _ provider.GenerateParams) (*provider.StreamResult, error) {
	ch := make(chan provider.StreamChunk, 2)
	ch <- provider.StreamChunk{Type: provider.ChunkText, Text: "hello"}
	ch <- provider.StreamChunk{Type: provider.ChunkFinish, FinishReason: provider.FinishStop}
	close(ch)
	return &provider.StreamResult{Stream: ch}, nil
}

func (m *mockLanguageModel) Capabilities() provider.ModelCapabilities { return m.caps }

// mockEmbeddingModel implements provider.EmbeddingModel for testing.
type mockEmbeddingModel struct {
	id string
}

func (m *mockEmbeddingModel) ModelID() string { return m.id }

func (m *mockEmbeddingModel) DoEmbed(_ context.Context, values []string, _ provider.EmbedParams) (*provider.EmbedResult, error) {
	embeddings := make([][]float64, len(values))
	for i := range values {
		embeddings[i] = []float64{0.1, 0.2, 0.3}
	}
	return &provider.EmbedResult{Embeddings: embeddings}, nil
}

func (m *mockEmbeddingModel) MaxValuesPerCall() int { return 100 }

// mockImageModel implements provider.ImageModel for testing.
type mockImageModel struct {
	id string
}

func (m *mockImageModel) ModelID() string { return m.id }

func (m *mockImageModel) DoGenerate(_ context.Context, _ provider.ImageParams) (*provider.ImageResult, error) {
	return &provider.ImageResult{
		Images: []provider.ImageData{{Data: []byte("fake-png"), MediaType: "image/png"}},
	}, nil
}

func TestLanguageModelInterface(t *testing.T) {
	var model provider.LanguageModel = &mockLanguageModel{
		id: "gpt-4o",
		caps: provider.ModelCapabilities{
			Temperature: true,
			ToolCall:    true,
			InputModalities: provider.ModalitySet{
				Text:  true,
				Image: true,
			},
			OutputModalities: provider.ModalitySet{Text: true},
		},
	}

	if model.ModelID() != "gpt-4o" {
		t.Errorf("ModelID() = %q, want %q", model.ModelID(), "gpt-4o")
	}

	caps := provider.ModelCapabilitiesOf(model)
	if !caps.Temperature || !caps.ToolCall {
		t.Error("expected Temperature and ToolCall capabilities")
	}
	if !caps.InputModalities.Image {
		t.Error("expected Image input modality")
	}

	// DoGenerate
	result, err := model.DoGenerate(t.Context(), provider.GenerateParams{
		Messages: []provider.Message{
			{Role: provider.RoleUser, Content: []provider.Part{{Type: provider.PartText, Text: "hi"}}},
		},
	})
	if err != nil {
		t.Fatalf("DoGenerate error: %v", err)
	}
	if result.Text != "hello" {
		t.Errorf("Text = %q, want %q", result.Text, "hello")
	}
	if result.FinishReason != provider.FinishStop {
		t.Errorf("FinishReason = %q, want %q", result.FinishReason, provider.FinishStop)
	}
	if result.Usage.InputTokens != 10 || result.Usage.OutputTokens != 5 {
		t.Errorf("Usage = %+v, want InputTokens=10, OutputTokens=5", result.Usage)
	}

	// DoStream
	stream, err := model.DoStream(t.Context(), provider.GenerateParams{
		Messages: []provider.Message{
			{Role: provider.RoleUser, Content: []provider.Part{{Type: provider.PartText, Text: "hi"}}},
		},
	})
	if err != nil {
		t.Fatalf("DoStream error: %v", err)
	}

	var chunks []provider.StreamChunk
	for chunk := range stream.Stream {
		chunks = append(chunks, chunk)
	}
	if len(chunks) != 2 {
		t.Fatalf("got %d chunks, want 2", len(chunks))
	}
	if chunks[0].Type != provider.ChunkText || chunks[0].Text != "hello" {
		t.Errorf("chunk[0] = %+v, want text chunk with 'hello'", chunks[0])
	}
	if chunks[1].Type != provider.ChunkFinish {
		t.Errorf("chunk[1].Type = %q, want %q", chunks[1].Type, provider.ChunkFinish)
	}
}

func TestEmbeddingModelInterface(t *testing.T) {
	var model provider.EmbeddingModel = &mockEmbeddingModel{id: "text-embedding-3-small"}

	if model.ModelID() != "text-embedding-3-small" {
		t.Errorf("ModelID() = %q, want %q", model.ModelID(), "text-embedding-3-small")
	}
	if model.MaxValuesPerCall() != 100 {
		t.Errorf("MaxValuesPerCall() = %d, want 100", model.MaxValuesPerCall())
	}

	result, err := model.DoEmbed(t.Context(), []string{"hello", "world"}, provider.EmbedParams{})
	if err != nil {
		t.Fatalf("DoEmbed error: %v", err)
	}
	if len(result.Embeddings) != 2 {
		t.Fatalf("got %d embeddings, want 2", len(result.Embeddings))
	}
}

func TestImageModelInterface(t *testing.T) {
	var model provider.ImageModel = &mockImageModel{id: "dall-e-3"}

	if model.ModelID() != "dall-e-3" {
		t.Errorf("ModelID() = %q, want %q", model.ModelID(), "dall-e-3")
	}

	result, err := model.DoGenerate(t.Context(), provider.ImageParams{
		Prompt: "a cat", N: 1, Size: "1024x1024",
	})
	if err != nil {
		t.Fatalf("DoGenerate error: %v", err)
	}
	if len(result.Images) != 1 {
		t.Fatalf("got %d images, want 1", len(result.Images))
	}
	if result.Images[0].MediaType != "image/png" {
		t.Errorf("MediaType = %q, want %q", result.Images[0].MediaType, "image/png")
	}
}

func TestGenerateParamsWithTools(t *testing.T) {
	schema := json.RawMessage(`{"type":"object","properties":{"path":{"type":"string"}}}`)
	params := provider.GenerateParams{
		Messages: []provider.Message{
			{Role: provider.RoleUser, Content: []provider.Part{{Type: provider.PartText, Text: "read file.go"}}},
		},
		System: "You are a coding assistant.",
		Tools: []provider.ToolDefinition{
			{Name: "read_file", Description: "Read a file", InputSchema: schema},
		},
		ToolChoice: "auto",
	}

	if len(params.Tools) != 1 {
		t.Fatalf("got %d tools, want 1", len(params.Tools))
	}
	if params.Tools[0].Name != "read_file" {
		t.Errorf("tool name = %q, want %q", params.Tools[0].Name, "read_file")
	}
}

func TestToolCallResult(t *testing.T) {
	tc := provider.ToolCall{
		ID:    "call_123",
		Name:  "read_file",
		Input: json.RawMessage(`{"path":"main.go"}`),
	}

	if tc.ID != "call_123" {
		t.Errorf("ID = %q, want %q", tc.ID, "call_123")
	}

	var input struct{ Path string }
	if err := json.Unmarshal(tc.Input, &input); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if input.Path != "main.go" {
		t.Errorf("Path = %q, want %q", input.Path, "main.go")
	}
}

func TestMessageParts(t *testing.T) {
	msg := provider.Message{
		Role: provider.RoleAssistant,
		Content: []provider.Part{
			{Type: provider.PartText, Text: "Let me read that file."},
			{
				Type:       provider.PartToolCall,
				ToolCallID: "call_1",
				ToolName:   "read_file",
				ToolInput:  json.RawMessage(`{"path":"main.go"}`),
			},
		},
	}

	if msg.Role != provider.RoleAssistant {
		t.Errorf("Role = %q, want %q", msg.Role, provider.RoleAssistant)
	}
	if len(msg.Content) != 2 {
		t.Fatalf("got %d parts, want 2", len(msg.Content))
	}
	if msg.Content[0].Type != provider.PartText {
		t.Errorf("part[0].Type = %q, want %q", msg.Content[0].Type, provider.PartText)
	}
	if msg.Content[1].Type != provider.PartToolCall {
		t.Errorf("part[1].Type = %q, want %q", msg.Content[1].Type, provider.PartToolCall)
	}
}

func TestStreamChunkTypes(t *testing.T) {
	types := []provider.StreamChunkType{
		provider.ChunkText,
		provider.ChunkReasoning,
		provider.ChunkToolCall,
		provider.ChunkToolCallDelta,
		provider.ChunkToolCallStreamStart,
		provider.ChunkToolResult,
		provider.ChunkStepFinish,
		provider.ChunkFinish,
		provider.ChunkError,
	}
	if len(types) != 9 {
		t.Errorf("expected 9 chunk types, got %d", len(types))
	}
}

func TestFinishReasons(t *testing.T) {
	reasons := []provider.FinishReason{
		provider.FinishStop,
		provider.FinishToolCalls,
		provider.FinishLength,
		provider.FinishContentFilter,
		provider.FinishError,
		provider.FinishOther,
	}
	if len(reasons) != 6 {
		t.Errorf("expected 6 finish reasons, got %d", len(reasons))
	}
}

func TestStaticToken(t *testing.T) {
	ts := provider.StaticToken("sk-test-key")

	tok, err := ts.Token(t.Context())
	if err != nil {
		t.Fatalf("Token error: %v", err)
	}
	if tok != "sk-test-key" {
		t.Errorf("Token = %q, want %q", tok, "sk-test-key")
	}

	// Repeated calls return the same value.
	tok2, _ := ts.Token(t.Context())
	if tok2 != tok {
		t.Error("StaticToken returned different values")
	}
}

func TestCachedTokenSource(t *testing.T) {
	callCount := 0
	ts := provider.CachedTokenSource(func(_ context.Context) (*provider.Token, error) {
		callCount++
		return &provider.Token{
			Value:     "token-v" + string(rune('0'+callCount)),
			ExpiresAt: time.Now().Add(time.Hour),
		}, nil
	})

	// First call fetches.
	tok, err := ts.Token(t.Context())
	if err != nil {
		t.Fatalf("Token error: %v", err)
	}
	if tok != "token-v1" {
		t.Errorf("Token = %q, want %q", tok, "token-v1")
	}
	if callCount != 1 {
		t.Errorf("callCount = %d, want 1", callCount)
	}

	// Second call returns cached.
	tok, _ = ts.Token(t.Context())
	if tok != "token-v1" {
		t.Errorf("Token = %q, want %q (cached)", tok, "token-v1")
	}
	if callCount != 1 {
		t.Errorf("callCount = %d, want 1 (should be cached)", callCount)
	}
}

func TestCachedTokenSourceExpiry(t *testing.T) {
	callCount := 0
	ts := provider.CachedTokenSource(func(_ context.Context) (*provider.Token, error) {
		callCount++
		return &provider.Token{
			Value:     "tok",
			ExpiresAt: time.Now().Add(-time.Second), // already expired
		}, nil
	})

	// First call fetches.
	_, _ = ts.Token(t.Context())
	if callCount != 1 {
		t.Fatalf("callCount = %d, want 1", callCount)
	}

	// Second call re-fetches because token is expired.
	_, _ = ts.Token(t.Context())
	if callCount != 2 {
		t.Errorf("callCount = %d, want 2 (token expired, should re-fetch)", callCount)
	}
}

func TestCachedTokenSourceInvalidate(t *testing.T) {
	callCount := 0
	ts := provider.CachedTokenSource(func(_ context.Context) (*provider.Token, error) {
		callCount++
		return &provider.Token{
			Value:     "tok",
			ExpiresAt: time.Now().Add(time.Hour),
		}, nil
	})

	_, _ = ts.Token(t.Context())
	if callCount != 1 {
		t.Fatalf("callCount = %d, want 1", callCount)
	}

	// Invalidate forces re-fetch.
	inv, ok := ts.(provider.InvalidatingTokenSource)
	if !ok {
		t.Fatal("CachedTokenSource should implement InvalidatingTokenSource")
	}
	inv.Invalidate()

	_, _ = ts.Token(t.Context())
	if callCount != 2 {
		t.Errorf("callCount = %d, want 2 (invalidated, should re-fetch)", callCount)
	}
}

func TestCachedTokenSourceNoExpiry(t *testing.T) {
	callCount := 0
	ts := provider.CachedTokenSource(func(_ context.Context) (*provider.Token, error) {
		callCount++
		return &provider.Token{Value: "forever"}, nil // zero ExpiresAt
	})

	// Zero ExpiresAt means "no expiry" -- token is cached indefinitely.
	_, _ = ts.Token(t.Context())
	_, _ = ts.Token(t.Context())
	if callCount != 1 {
		t.Errorf("callCount = %d, want 1 (zero ExpiresAt should cache forever)", callCount)
	}
}

func TestCachedTokenSourceFetchError(t *testing.T) {
	errFetch := fmt.Errorf("auth server down")
	ts := provider.CachedTokenSource(func(_ context.Context) (*provider.Token, error) {
		return nil, errFetch
	})

	tok, err := ts.Token(t.Context())
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if tok != "" {
		t.Errorf("Token = %q, want empty on error", tok)
	}
	if err != errFetch {
		t.Errorf("err = %v, want %v", err, errFetch)
	}
}

// bareLanguageModel implements LanguageModel but NOT CapableModel.
type bareLanguageModel struct {
	id string
}

func (m *bareLanguageModel) ModelID() string { return m.id }

func (m *bareLanguageModel) DoGenerate(_ context.Context, _ provider.GenerateParams) (*provider.GenerateResult, error) {
	return &provider.GenerateResult{Text: "ok"}, nil
}

func (m *bareLanguageModel) DoStream(_ context.Context, _ provider.GenerateParams) (*provider.StreamResult, error) {
	ch := make(chan provider.StreamChunk)
	close(ch)
	return &provider.StreamResult{Stream: ch}, nil
}

func TestModelCapabilitiesOf_NonCapableModel(t *testing.T) {
	model := &bareLanguageModel{id: "bare-model"}
	caps := provider.ModelCapabilitiesOf(model)

	// Should return zero-value capabilities.
	if caps.Temperature || caps.ToolCall || caps.InputModalities.Image || caps.OutputModalities.Text {
		t.Errorf("expected zero ModelCapabilities, got %+v", caps)
	}
}

func TestTrySend_Success(t *testing.T) {
	out := make(chan provider.StreamChunk, 1)
	chunk := provider.StreamChunk{Type: provider.ChunkText, Text: "hello"}
	if !provider.TrySend(t.Context(), out, chunk) {
		t.Fatal("TrySend returned false on active context")
	}
	got := <-out
	if got.Text != "hello" {
		t.Errorf("Text = %q, want %q", got.Text, "hello")
	}
}

func TestTrySend_CancelledContext(t *testing.T) {
	out := make(chan provider.StreamChunk) // unbuffered - would block
	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	chunk := provider.StreamChunk{Type: provider.ChunkText, Text: "hello"}
	if provider.TrySend(ctx, out, chunk) {
		t.Fatal("TrySend returned true on cancelled context")
	}
}

