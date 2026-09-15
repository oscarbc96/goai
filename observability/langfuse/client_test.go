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

// dropFirstServer answers like Langfuse ingestion but slams the socket shut
// on the first request without writing a response, which is what a client
// sees when it reuses a keep-alive connection the server already closed
// (NEU-1375). Every later request gets 207 and its batch is recorded.
func dropFirstServer(t *testing.T) (*httptest.Server, *int32, *[][]string) {
	t.Helper()
	var calls int32
	var batches [][]string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := atomic.AddInt32(&calls, 1)
		if n == 1 {
			hj, ok := w.(http.Hijacker)
			if !ok {
				t.Fatal("response writer is not a Hijacker")
			}
			conn, _, err := hj.Hijack()
			if err != nil {
				t.Fatalf("hijack: %v", err)
			}
			_ = conn.Close()
			return
		}
		body, _ := io.ReadAll(r.Body)
		var env struct {
			Batch []struct {
				ID string `json:"id"`
			} `json:"batch"`
		}
		_ = json.Unmarshal(body, &env)
		ids := make([]string, 0, len(env.Batch))
		for _, e := range env.Batch {
			ids = append(ids, e.ID)
		}
		batches = append(batches, ids)
		w.WriteHeader(http.StatusMultiStatus)
	}))
	t.Cleanup(srv.Close)
	return srv, &calls, &batches
}

func TestFlush_RetriesOnceWhenServerDropsConnection(t *testing.T) {
	srv, calls, batches := dropFirstServer(t)
	c := newClient(srv.URL, "pk", "sk")
	c.appendEvents([]ingestionEvent{{ID: "e1", Type: "trace-create"}, {ID: "e2", Type: "generation-create"}})

	if err := c.flush(context.Background()); err != nil {
		t.Fatalf("flush should succeed on the retry, got: %v", err)
	}
	if got := atomic.LoadInt32(calls); got != 2 {
		t.Fatalf("want exactly 2 requests (1 dropped + 1 retry), got %d", got)
	}
	if len(*batches) != 1 || len((*batches)[0]) != 2 || (*batches)[0][0] != "e1" || (*batches)[0][1] != "e2" {
		t.Fatalf("retry must resend the same batch with the same event ids: %v", *batches)
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if len(c.events) != 0 {
		t.Fatalf("flushed events must not be requeued: %v", c.events)
	}
}

func TestFlush_GivesUpAfterOneRetry(t *testing.T) {
	var calls int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		atomic.AddInt32(&calls, 1)
		conn, _, err := w.(http.Hijacker).Hijack()
		if err != nil {
			t.Fatalf("hijack: %v", err)
		}
		_ = conn.Close()
	}))
	t.Cleanup(srv.Close)
	c := newClient(srv.URL, "pk", "sk")
	c.appendEvents([]ingestionEvent{{ID: "e1"}})

	if err := c.flush(context.Background()); err == nil {
		t.Fatal("flush must report the failure when the retry dies too")
	}
	if got := atomic.LoadInt32(&calls); got != 2 {
		t.Fatalf("want 2 attempts total, never more, got %d", got)
	}
}

func TestFlush_DoesNotRetryHTTPErrors(t *testing.T) {
	var calls int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		atomic.AddInt32(&calls, 1)
		w.WriteHeader(http.StatusInternalServerError)
	}))
	t.Cleanup(srv.Close)
	c := newClient(srv.URL, "pk", "sk")
	c.appendEvents([]ingestionEvent{{ID: "e1"}})

	if err := c.flush(context.Background()); err == nil {
		t.Fatal("5xx must surface as an error")
	}
	if got := atomic.LoadInt32(&calls); got != 1 {
		t.Fatalf("HTTP status errors are not retried, got %d requests", got)
	}
}

func TestFlush_RespectsCancelledContext(t *testing.T) {
	var calls int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		atomic.AddInt32(&calls, 1)
		conn, _, _ := w.(http.Hijacker).Hijack()
		_ = conn.Close()
	}))
	t.Cleanup(srv.Close)
	c := newClient(srv.URL, "pk", "sk")
	c.appendEvents([]ingestionEvent{{ID: "e1"}})

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := c.flush(ctx); err == nil {
		t.Fatal("flush with a dead context must fail")
	}
	if got := atomic.LoadInt32(&calls); got != 0 {
		t.Fatalf("a cancelled context must not be retried, got %d requests", got)
	}
}

// Langfuse's Node server closes idle keep-alive sockets after 5 s; our pool
// must evict first, otherwise every post-LLM flush reuses a dead socket.
func TestTransport_EvictsIdleConnsBeforeLangfuseDoes(t *testing.T) {
	const langfuseKeepAliveTimeout = 5 * time.Second
	c := newClient("http://localhost", "pk", "sk")
	tr, ok := c.httpClient.Transport.(*http.Transport)
	if !ok {
		t.Fatalf("client transport is %T, want *http.Transport", c.httpClient.Transport)
	}
	if tr.IdleConnTimeout <= 0 || tr.IdleConnTimeout >= langfuseKeepAliveTimeout {
		t.Fatalf("IdleConnTimeout = %s, must be > 0 and < %s", tr.IdleConnTimeout, langfuseKeepAliveTimeout)
	}
	if tr.DisableKeepAlives {
		t.Fatal("keep-alives should stay on for the 500ms burst flushes")
	}
}
