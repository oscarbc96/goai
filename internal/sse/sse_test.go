package sse

import (
	"strings"
	"testing"
)

func TestScanner_BasicEvents(t *testing.T) {
	input := "data: hello\ndata: world\ndata: [DONE]\n"
	s := NewScanner(strings.NewReader(input))

	data, ok := s.Next()
	if !ok || data != "hello" {
		t.Errorf("first: got %q, %v; want %q, true", data, ok, "hello")
	}

	data, ok = s.Next()
	if !ok || data != "world" {
		t.Errorf("second: got %q, %v; want %q, true", data, ok, "world")
	}

	data, ok = s.Next()
	if ok {
		t.Errorf("after DONE: got %q, %v; want false", data, ok)
	}
	if !s.IsDone() {
		t.Error("IsDone should be true after [DONE]")
	}
}

func TestScanner_SkipsNonDataLines(t *testing.T) {
	input := "event: message\nid: 1\ndata: payload\n\nretry: 5000\ndata: [DONE]\n"
	s := NewScanner(strings.NewReader(input))

	data, ok := s.Next()
	if !ok || data != "payload" {
		t.Errorf("got %q, %v; want %q, true", data, ok, "payload")
	}

	data, ok = s.Next()
	if ok {
		t.Errorf("after DONE: got %q, %v; want false", data, ok)
	}
}

func TestScanner_EmptyStream(t *testing.T) {
	s := NewScanner(strings.NewReader(""))

	data, ok := s.Next()
	if ok {
		t.Errorf("empty stream: got %q, %v; want false", data, ok)
	}
	if s.IsDone() {
		t.Error("IsDone should be false for empty stream (no [DONE] seen)")
	}
}

func TestScanner_NoDataPrefix(t *testing.T) {
	input := "event: ping\n\nevent: pong\n"
	s := NewScanner(strings.NewReader(input))

	data, ok := s.Next()
	if ok {
		t.Errorf("no data lines: got %q, %v; want false", data, ok)
	}
}

func TestScanner_JSONPayloads(t *testing.T) {
	input := `data: {"id":"1","choices":[{"delta":{"content":"hi"}}]}
data: {"id":"2","choices":[{"delta":{"content":" there"}}]}
data: [DONE]
`
	s := NewScanner(strings.NewReader(input))

	data, ok := s.Next()
	if !ok {
		t.Fatal("expected first event")
	}
	if !strings.Contains(data, `"content":"hi"`) {
		t.Errorf("first event missing content: %s", data)
	}

	data, ok = s.Next()
	if !ok {
		t.Fatal("expected second event")
	}
	if !strings.Contains(data, `"content":" there"`) {
		t.Errorf("second event missing content: %s", data)
	}

	_, ok = s.Next()
	if ok {
		t.Error("expected false after DONE")
	}
}

func TestScanner_DoneIdempotent(t *testing.T) {
	input := "data: first\ndata: [DONE]\ndata: after-done\n"
	s := NewScanner(strings.NewReader(input))

	s.Next() // "first"
	s.Next() // DONE

	// Calling Next after DONE should keep returning false.
	for i := 0; i < 3; i++ {
		_, ok := s.Next()
		if ok {
			t.Errorf("call %d after DONE returned ok=true", i)
		}
	}
}

func TestScanner_Err(t *testing.T) {
	input := "data: ok\n"
	s := NewScanner(strings.NewReader(input))
	s.Next()
	s.Next() // EOF

	if err := s.Err(); err != nil {
		t.Errorf("Err() = %v, want nil", err)
	}
}
