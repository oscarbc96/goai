package langfuse

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// A flush against a Langfuse that does not answer must end at the bound and
// report it, not hold the caller for the client's 30s timeout. The final
// flush runs inside goai's GenerateObject on the error path, so on a Lambda
// that has just overrun its budget every second here is a second it does not
// have (NEU-1416).
func TestFlushBoundedEndsAStalledIngestion(t *testing.T) {
	release := make(chan struct{})
	srv := httptest.NewServer(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		select {
		case <-release:
		case <-r.Context().Done():
		}
	}))
	// Registered after srv.Close so it runs FIRST (cleanups are LIFO): Close
	// waits for the handler, which waits for release.
	t.Cleanup(srv.Close)
	t.Cleanup(func() { close(release) })

	lc := newClient(srv.URL, "pk", "sk")
	lc.appendEvents([]ingestionEvent{{ID: "1", Type: eventTrace, Body: traceBody{ID: "t"}}})

	var got error
	cfg := Config{FlushTimeout: 100 * time.Millisecond, OnFlushError: func(err error) { got = err }}

	start := time.Now()
	flushBounded(context.Background(), cfg, lc)
	elapsed := time.Since(start)

	if got == nil {
		t.Fatal("a flush that never completes must be reported through OnFlushError")
	}
	if elapsed > 5*time.Second {
		t.Errorf("flush was not bounded: took %s", elapsed)
	}
	if !strings.Contains(got.Error(), "exceeded its 100ms bound") {
		t.Errorf("error should name the bound, got: %v", got)
	}
	if !errors.Is(got, context.DeadlineExceeded) {
		t.Errorf("error should keep the deadline cause for callers, got: %v", got)
	}
}

// Zero means the default, never "no bound".
func TestFlushBoundDefaults(t *testing.T) {
	if got := (Config{}).flushBound(); got != DefaultFlushTimeout {
		t.Errorf("flushBound() = %v, want DefaultFlushTimeout %v", got, DefaultFlushTimeout)
	}
	if got := (Config{FlushTimeout: 3 * time.Second}).flushBound(); got != 3*time.Second {
		t.Errorf("flushBound() = %v, want 3s", got)
	}
}
