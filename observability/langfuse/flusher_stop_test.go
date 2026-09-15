package langfuse

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/zendev-sh/goai"
	"github.com/zendev-sh/goai/provider"
)

// stubModel answers once, but only after the background flusher has a POST in
// flight, so the run ends while that flush is still running.
type stubModel struct{ inFlight <-chan struct{} }

func (m stubModel) ModelID() string { return "stub" }

func (m stubModel) DoGenerate(ctx context.Context, _ provider.GenerateParams) (*provider.GenerateResult, error) {
	select {
	case <-m.inFlight:
	case <-ctx.Done():
		return nil, ctx.Err()
	}
	return &provider.GenerateResult{Text: `{"ok":true}`, FinishReason: provider.FinishStop}, nil
}

func (m stubModel) DoStream(context.Context, provider.GenerateParams) (*provider.StreamResult, error) {
	panic("not used")
}

// The run must end even when it finishes while a ticker flush is in flight.
// end() used to close flushStop and then set the shared variable to nil; the
// flusher re-read that variable on its next select, got a nil channel that
// never becomes ready, kept ticking forever, and end()'s flushWG.Wait() never
// returned. Nothing bounds a WaitGroup, so the run's context deadline could
// not end it either: the ticket-orchestrator Lambda sat silent until its 180s
// cap (NEU-1533).
func TestRunEndsWhileATickerFlushIsInFlight(t *testing.T) {
	inFlight := make(chan struct{})
	var once sync.Once
	release := make(chan struct{})
	srv := httptest.NewServer(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		once.Do(func() { close(inFlight) })
		select {
		case <-release:
		case <-r.Context().Done():
		}
	}))
	t.Cleanup(srv.Close)
	t.Cleanup(func() { close(release) })

	h := New(Config{PublicKey: "pk", SecretKey: "sk", Host: srv.URL, FlushTimeout: 300 * time.Millisecond})

	done := make(chan error, 1)
	go func() {
		_, err := goai.GenerateObject[map[string]any](context.Background(), stubModel{inFlight: inFlight},
			append(h.Run(), goai.WithPrompt("hi"))...)
		done <- err
	}()

	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("GenerateObject: %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("run did not end: end() is stuck waiting for a flusher that never stops")
	}
}
