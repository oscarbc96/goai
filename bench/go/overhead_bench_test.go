package bench

import (
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/zendev-sh/goai"
	"github.com/zendev-sh/goai/provider/openai"
)

// BenchmarkOverheadRawHTTP measures raw net/http against the mock server (baseline).
func BenchmarkOverheadRawHTTP(b *testing.B) {
	srv := NewMockServer()
	defer srv.Close()

	client := &http.Client{}
	body := `{"model":"gpt-4o","messages":[{"role":"user","content":"bench"}],"stream":true}`

	b.ResetTimer()
	b.ReportAllocs()
	for b.Loop() {
		req, _ := http.NewRequest("POST", srv.URL+"/v1/chat/completions", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		resp, err := client.Do(req)
		if err != nil {
			b.Fatal(err)
		}
		io.Copy(io.Discard, resp.Body)
		resp.Body.Close()
	}
}

// BenchmarkOverheadGoAI measures GoAI SDK streaming against the mock server.
func BenchmarkOverheadGoAI(b *testing.B) {
	srv := NewMockServer()
	defer srv.Close()

	model := openai.Chat("gpt-4o", openai.WithAPIKey("bench"), openai.WithBaseURL(srv.URL+"/v1"))
	ctx := b.Context()
	opts := goai.WithProviderOptions(map[string]any{"useResponsesAPI": false})

	b.ResetTimer()
	b.ReportAllocs()
	for b.Loop() {
		stream, err := goai.StreamText(ctx, model, goai.WithPrompt("bench"), opts)
		if err != nil {
			b.Fatal(err)
		}
		for range stream.TextStream() {
		}
	}
}

// BenchmarkOverheadGenerateText measures non-streaming GoAI vs raw HTTP.
func BenchmarkOverheadGenerateText(b *testing.B) {
	srv := NewMockServer()
	defer srv.Close()

	model := openai.Chat("gpt-4o", openai.WithAPIKey("bench"), openai.WithBaseURL(srv.URL+"/v1"))
	ctx := b.Context()
	opts := goai.WithProviderOptions(map[string]any{"useResponsesAPI": false})

	b.ResetTimer()
	b.ReportAllocs()
	for b.Loop() {
		_, err := goai.GenerateText(ctx, model, goai.WithPrompt("bench"), opts)
		if err != nil {
			b.Fatal(err)
		}
	}
}
