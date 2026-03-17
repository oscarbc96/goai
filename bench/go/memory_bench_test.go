package bench

import (
	"runtime"
	"sync"
	"testing"

	"github.com/zendev-sh/goai"
	"github.com/zendev-sh/goai/provider/openai"
)

func measureHeap() uint64 {
	runtime.GC()
	runtime.GC()
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	return m.HeapAlloc
}

// BenchmarkMemory1Stream measures heap usage for a single stream.
func BenchmarkMemory1Stream(b *testing.B) {
	benchmarkMemoryStreams(b, 1)
}

// BenchmarkMemory10Streams measures heap usage for 10 concurrent streams.
func BenchmarkMemory10Streams(b *testing.B) {
	benchmarkMemoryStreams(b, 10)
}

// BenchmarkMemory50Streams measures heap usage for 50 concurrent streams.
func BenchmarkMemory50Streams(b *testing.B) {
	benchmarkMemoryStreams(b, 50)
}

// BenchmarkMemory100Streams measures heap usage for 100 concurrent streams.
func BenchmarkMemory100Streams(b *testing.B) {
	benchmarkMemoryStreams(b, 100)
}

// benchmarkMemoryStreams creates N streams sequentially (matching TS benchmark),
// reads the first chunk from each to establish the stream, measures heap delta,
// then drains all streams.
func benchmarkMemoryStreams(b *testing.B, count int) {
	srv := NewMockServer()
	defer srv.Close()

	model := openai.Chat("gpt-4o", openai.WithAPIKey("bench"), openai.WithBaseURL(srv.URL+"/v1"))
	ctx := b.Context()
	opts := goai.WithProviderOptions(map[string]any{"useResponsesAPI": false})

	b.ResetTimer()
	for b.Loop() {
		before := measureHeap()

		// Create streams sequentially (same as TS benchmark).
		streams := make([]*goai.TextStream, count)
		for j := range count {
			s, err := goai.StreamText(ctx, model, goai.WithPrompt("bench"), opts)
			if err != nil {
				b.Fatal(err)
			}
			streams[j] = s
			// Read first chunk to establish the stream.
			for range s.TextStream() {
				break
			}
		}

		after := measureHeap()
		heapDelta := after - before

		// Drain all streams after measurement.
		var wg sync.WaitGroup
		wg.Add(count)
		for _, s := range streams {
			go func(s *goai.TextStream) {
				defer wg.Done()
				for range s.TextStream() {
				}
			}(s)
		}
		wg.Wait()

		b.ReportMetric(float64(heapDelta), "heap-bytes/op")
	}
}
