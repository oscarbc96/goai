package bench

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

// fixturesDir returns the absolute path to bench/fixtures/.
func fixturesDir() string {
	_, f, _, _ := runtime.Caller(0)
	return filepath.Join(filepath.Dir(f), "..", "fixtures")
}

// NewMockServer creates an httptest.Server serving Chat Completions API fixtures.
// Both Go and TS benchmarks use Chat Completions API for fair comparison
// (same SSE format, same fixtures, same parse workload).
//
// Routes:
//   POST /v1/chat/completions (stream:true)  → stream_100x500.jsonl
//   POST /v1/chat/completions (stream:false) → generate_single.json
func NewMockServer() *httptest.Server {
	streamData, err := os.ReadFile(filepath.Join(fixturesDir(), "stream_100x500.jsonl"))
	if err != nil {
		panic("missing fixture: " + err.Error())
	}
	singleData, err := os.ReadFile(filepath.Join(fixturesDir(), "generate_single.json"))
	if err != nil {
		panic("missing fixture: " + err.Error())
	}

	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Read body to detect stream parameter.
		body := make([]byte, 4096)
		n, _ := r.Body.Read(body)
		bodyStr := string(body[:n])

		if strings.Contains(bodyStr, `"stream":true`) || strings.Contains(bodyStr, `"stream": true`) {
			w.Header().Set("Content-Type", "text/event-stream")
			w.Header().Set("Cache-Control", "no-cache")
			w.Header().Set("Connection", "keep-alive")
			w.Write(streamData)
		} else {
			w.Header().Set("Content-Type", "application/json")
			w.Write(singleData)
		}
	}))
}
