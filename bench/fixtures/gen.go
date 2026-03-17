//go:build ignore

// gen.go generates SSE fixture files for benchmarking.
package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

func main() {
	generateStreamFixture()
	generateSingleFixture()
	fmt.Println("fixtures generated")
}

func generateStreamFixture() {
	f, _ := os.Create("stream_100x500.jsonl")
	defer f.Close()

	// 500 bytes of text content per chunk
	text := strings.Repeat("Lorem ipsum dolor sit amet, consectetur adipiscing elit. ", 9)
	text = text[:500]

	for i := 0; i < 100; i++ {
		chunk := map[string]any{
			"id":      fmt.Sprintf("chatcmpl-%d", i),
			"object":  "chat.completion.chunk",
			"created": 1700000000,
			"model":   "gpt-4o",
			"choices": []map[string]any{
				{
					"index": 0,
					"delta": map[string]any{
						"content": text,
					},
				},
			},
		}
		data, _ := json.Marshal(chunk)
		fmt.Fprintf(f, "data: %s\n\n", data)
	}

	// Final chunk with finish_reason
	final := map[string]any{
		"id":      "chatcmpl-final",
		"object":  "chat.completion.chunk",
		"created": 1700000000,
		"model":   "gpt-4o",
		"choices": []map[string]any{
			{
				"index":         0,
				"delta":         map[string]any{},
				"finish_reason": "stop",
			},
		},
		"usage": map[string]any{
			"prompt_tokens":     10,
			"completion_tokens": 100,
			"total_tokens":      110,
		},
	}
	data, _ := json.Marshal(final)
	fmt.Fprintf(f, "data: %s\n\n", data)
	fmt.Fprint(f, "data: [DONE]\n\n")
}

func generateSingleFixture() {
	text := strings.Repeat("Lorem ipsum dolor sit amet, consectetur adipiscing elit. ", 9)
	text = text[:500]

	resp := map[string]any{
		"id":      "chatcmpl-single",
		"object":  "chat.completion",
		"created": 1700000000,
		"model":   "gpt-4o",
		"choices": []map[string]any{
			{
				"index": 0,
				"message": map[string]any{
					"role":    "assistant",
					"content": text,
				},
				"finish_reason": "stop",
			},
		},
		"usage": map[string]any{
			"prompt_tokens":     10,
			"completion_tokens": 100,
			"total_tokens":      110,
		},
	}
	data, _ := json.MarshalIndent(resp, "", "  ")
	os.WriteFile("generate_single.json", data, 0644)
}
