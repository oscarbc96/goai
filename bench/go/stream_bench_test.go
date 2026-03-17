package bench

import (
	"testing"

	"github.com/zendev-sh/goai"
	"github.com/zendev-sh/goai/provider/openai"
)

// chatCompletionsOpts forces Chat Completions API (not Responses API)
// so both Go and TS benchmarks use identical SSE fixtures.
var chatCompletionsOpts = goai.WithProviderOptions(map[string]any{"useResponsesAPI": false})

// BenchmarkStreamingThroughput measures total time to open + consume a 100-chunk SSE stream.
func BenchmarkStreamingThroughput(b *testing.B) {
	srv := NewMockServer()
	defer srv.Close()

	model := openai.Chat("gpt-4o", openai.WithAPIKey("bench"), openai.WithBaseURL(srv.URL+"/v1"))
	ctx := b.Context()

	b.ResetTimer()
	b.ReportAllocs()
	for b.Loop() {
		stream, err := goai.StreamText(ctx, model, goai.WithPrompt("bench"), chatCompletionsOpts)
		if err != nil {
			b.Fatal(err)
		}
		for range stream.TextStream() {
		}
	}
}

// BenchmarkTimeToFirstChunk measures latency from StreamText() call to first text chunk received.
// IMPORTANT: only the time up to the first chunk is measured.
// The remaining stream is drained outside the timed region.
func BenchmarkTimeToFirstChunk(b *testing.B) {
	srv := NewMockServer()
	defer srv.Close()

	model := openai.Chat("gpt-4o", openai.WithAPIKey("bench"), openai.WithBaseURL(srv.URL+"/v1"))
	ctx := b.Context()

	b.ResetTimer()
	b.ReportAllocs()
	for b.Loop() {
		stream, err := goai.StreamText(ctx, model, goai.WithPrompt("bench"), chatCompletionsOpts)
		if err != nil {
			b.Fatal(err)
		}
		// Time only includes up to the first chunk.
		for range stream.TextStream() {
			break
		}
		b.StopTimer()
		// Drain remaining chunks outside timed region.
		for range stream.TextStream() {
		}
		b.StartTimer()
	}
}

// BenchmarkConcurrentStreams measures throughput with multiple concurrent streams.
func BenchmarkConcurrentStreams(b *testing.B) {
	srv := NewMockServer()
	defer srv.Close()

	model := openai.Chat("gpt-4o", openai.WithAPIKey("bench"), openai.WithBaseURL(srv.URL+"/v1"))
	ctx := b.Context()

	b.ResetTimer()
	b.ReportAllocs()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			stream, err := goai.StreamText(ctx, model, goai.WithPrompt("bench"), chatCompletionsOpts)
			if err != nil {
				b.Fatal(err)
			}
			for range stream.TextStream() {
			}
		}
	})
}
