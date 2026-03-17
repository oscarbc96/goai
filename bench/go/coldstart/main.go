// coldstart measures GoAI import + first call latency.
// Run: go build -o coldstart . && for i in $(seq 20); do ./coldstart; done
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"time"

	"github.com/zendev-sh/goai"
	"github.com/zendev-sh/goai/provider/openai"
)

func main() {
	start := time.Now()

	// Minimal mock server returning Chat Completions format.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"id":"chatcmpl-bench","object":"chat.completion","created":1700000000,"model":"gpt-4o","choices":[{"index":0,"message":{"role":"assistant","content":"ok"},"finish_reason":"stop"}],"usage":{"prompt_tokens":1,"completion_tokens":1,"total_tokens":2}}`)
	}))
	defer srv.Close()

	model := openai.Chat("gpt-4o", openai.WithAPIKey("bench"), openai.WithBaseURL(srv.URL+"/v1"))
	opts := goai.WithProviderOptions(map[string]any{"useResponsesAPI": false})
	_, err := goai.GenerateText(context.Background(), model, goai.WithPrompt("hi"), opts)
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}

	elapsed := time.Since(start)
	out, _ := json.Marshal(map[string]any{
		"benchmark": "cold_start",
		"ns":        elapsed.Nanoseconds(),
	})
	fmt.Println(string(out))
}
