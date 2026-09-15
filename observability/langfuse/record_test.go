package langfuse

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"
)

type capturedEvent struct {
	ID   string          `json:"id"`
	Type string          `json:"type"`
	Body json.RawMessage `json:"body"`
}

func TestRecordTrace_SendsTraceRootSpanAndChildren(t *testing.T) {
	var batch []capturedEvent
	var auth string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		auth = r.Header.Get("Authorization")
		body, _ := io.ReadAll(r.Body)
		var env struct {
			Batch []capturedEvent `json:"batch"`
		}
		_ = json.Unmarshal(body, &env)
		batch = env.Batch
		w.WriteHeader(http.StatusMultiStatus)
	}))
	t.Cleanup(srv.Close)

	start := time.Date(2026, 9, 14, 10, 0, 0, 0, time.UTC)
	traceID, err := RecordTrace(context.Background(), TraceRecord{
		Start:  start,
		End:    start.Add(5 * time.Millisecond),
		Input:  map[string]any{"event": "x"},
		Output: map[string]any{"rule": "rule_3"},
		Spans:  []SpanRecord{{Name: "deterministic-router", Metadata: map[string]any{"rule": "rule_3"}}},
	},
		PublicKey("pk"), SecretKey("sk"), Host(srv.URL),
		TraceName("ticket-orchestrator"), SessionID("ticket-1"), UserID("org-1"),
		Tags("router:deterministic"), Metadata(map[string]any{"router": "deterministic"}),
		Version("1.10.0"), Environment("default"),
	)
	if err != nil {
		t.Fatalf("RecordTrace: %v", err)
	}
	if traceID == "" || auth == "" {
		t.Fatalf("traceID=%q auth=%q", traceID, auth)
	}
	if len(batch) != 3 {
		t.Fatalf("want trace + root span + 1 child, got %d events", len(batch))
	}

	var trace traceBody
	_ = json.Unmarshal(batch[0].Body, &trace)
	if batch[0].Type != eventTrace || trace.ID != traceID || trace.Name != "ticket-orchestrator" ||
		trace.SessionID != "ticket-1" || trace.UserID != "org-1" || trace.Version != "1.10.0" {
		t.Errorf("trace = %+v (%s)", trace, batch[0].Type)
	}
	meta, _ := trace.Metadata.(map[string]any)
	if meta["router"] != "deterministic" || meta["environment"] != "default" {
		t.Errorf("trace metadata = %v", trace.Metadata)
	}

	var root, child spanBody
	_ = json.Unmarshal(batch[1].Body, &root)
	_ = json.Unmarshal(batch[2].Body, &child)
	if root.TraceID != traceID || root.Name != "ticket-orchestrator" || root.ParentObservationID != "" || root.EndTime == "" {
		t.Errorf("root span = %+v", root)
	}
	if child.TraceID != traceID || child.ParentObservationID != root.ID || child.Name != "deterministic-router" {
		t.Errorf("child span = %+v", child)
	}
	seen := map[string]bool{}
	for _, e := range batch {
		if seen[e.ID] {
			t.Errorf("duplicate envelope id %s", e.ID)
		}
		seen[e.ID] = true
	}
}

func TestRecordTrace_NoCredentialsIsANoop(t *testing.T) {
	t.Setenv("LANGFUSE_PUBLIC_KEY", "")
	t.Setenv("LANGFUSE_SECRET_KEY", "")
	var calls int32
	srv := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) { atomic.AddInt32(&calls, 1) }))
	t.Cleanup(srv.Close)

	id, err := RecordTrace(context.Background(), TraceRecord{}, Host(srv.URL))
	if id != "" || err != nil || atomic.LoadInt32(&calls) != 0 {
		t.Fatalf("id=%q err=%v calls=%d", id, err, calls)
	}
}

func TestRecordTrace_ReturnsAndReportsFlushFailure(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	t.Cleanup(srv.Close)
	var reported error
	_, err := RecordTrace(context.Background(), TraceRecord{},
		PublicKey("pk"), SecretKey("sk"), Host(srv.URL), OnFlushError(func(e error) { reported = e }))
	if err == nil || reported == nil {
		t.Fatalf("err=%v reported=%v", err, reported)
	}
}

// A cancelled run context must not cost the trace: the flush is detached.
func TestRecordTrace_SurvivesACancelledContext(t *testing.T) {
	var calls int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		atomic.AddInt32(&calls, 1)
		w.WriteHeader(http.StatusMultiStatus)
	}))
	t.Cleanup(srv.Close)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := RecordTrace(ctx, TraceRecord{}, PublicKey("pk"), SecretKey("sk"), Host(srv.URL)); err != nil {
		t.Fatalf("RecordTrace: %v", err)
	}
	if atomic.LoadInt32(&calls) != 1 {
		t.Fatalf("calls = %d", calls)
	}
}
