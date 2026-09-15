package langfuse

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
)

// respondWith serves every ingestion request with the given status and body,
// counting requests.
func respondWith(t *testing.T, status int, body string) (*httptest.Server, *int32) {
	t.Helper()
	var calls int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		atomic.AddInt32(&calls, 1)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		_, _ = w.Write([]byte(body))
	}))
	t.Cleanup(srv.Close)
	return srv, &calls
}

// The response Langfuse 3.225.1 actually returned for a batch of one valid and
// one invalid event (error detail shortened).
const partial207 = `{"successes":[{"id":"ok-1","status":201}],"errors":[{"id":"bad-1","status":400,"message":"Invalid request data","error":"[\n {\"code\": \"invalid_format\", \"path\": [\"timestamp\"], \"message\": \"Invalid ISO datetime\"}\n]"}]}`

func TestFlush_ReportsEventsRejectedInsideA207(t *testing.T) {
	srv, calls := respondWith(t, http.StatusMultiStatus, partial207)
	c := newClient(srv.URL, "pk", "sk")
	c.appendEvents([]ingestionEvent{{ID: "ok-1", Type: eventTrace}, {ID: "bad-1", Type: eventSpan}})

	err := c.flush(context.Background())
	var rej *IngestionRejectedError
	if !errors.As(err, &rej) {
		t.Fatalf("want *IngestionRejectedError, got %v", err)
	}
	if rej.Accepted != 1 || len(rej.Rejected) != 1 || rej.Rejected[0].ID != "bad-1" || rej.Rejected[0].Status != 400 {
		t.Errorf("rejection = %+v", rej)
	}
	msg := err.Error()
	for _, want := range []string{"rejected 1 of 2 events", "bad-1", "status 400", "Invalid request data", "Invalid ISO datetime"} {
		if !strings.Contains(msg, want) {
			t.Errorf("error %q does not mention %q", msg, want)
		}
	}
	if strings.Contains(msg, "\n") {
		t.Errorf("error must be one line for log readers: %q", msg)
	}
	if atomic.LoadInt32(calls) != 1 {
		t.Errorf("a rejected event is rejected for its content and must not be resent; calls = %d", *calls)
	}
}

func TestFlush_A207WithOnlySuccessesIsSuccess(t *testing.T) {
	srv, _ := respondWith(t, http.StatusMultiStatus, `{"successes":[{"id":"a","status":201}],"errors":[]}`)
	c := newClient(srv.URL, "pk", "sk")
	c.appendEvents([]ingestionEvent{{ID: "a", Type: eventTrace}})
	if err := c.flush(context.Background()); err != nil {
		t.Fatalf("flush: %v", err)
	}
}

// A body that is empty or not the documented shape says nothing about
// rejection; the status already said the batch was accepted.
func TestFlush_UnrecognisedSuccessBodyIsSuccess(t *testing.T) {
	for _, body := range []string{"", "not json", `{"unexpected":true}`} {
		srv, _ := respondWith(t, http.StatusMultiStatus, body)
		c := newClient(srv.URL, "pk", "sk")
		c.appendEvents([]ingestionEvent{{ID: "a", Type: eventTrace}})
		if err := c.flush(context.Background()); err != nil {
			t.Errorf("body %q: flush: %v", body, err)
		}
	}
}

func TestFlush_HTTPErrorIncludesTheResponse(t *testing.T) {
	srv, _ := respondWith(t, http.StatusUnauthorized, `{"message":"Invalid public key"}`)
	c := newClient(srv.URL, "pk", "sk")
	c.appendEvents([]ingestionEvent{{ID: "a", Type: eventTrace}})
	err := c.flush(context.Background())
	if err == nil || !strings.Contains(err.Error(), "status 401") || !strings.Contains(err.Error(), "Invalid public key") {
		t.Fatalf("err = %v", err)
	}
}

func TestIngestionRejectedError_CapsTheListing(t *testing.T) {
	e := &IngestionRejectedError{Accepted: 2, Rejected: []RejectedEvent{
		{ID: "e1", Status: 400}, {ID: "e2", Status: 400}, {ID: "e3", Status: 400}, {ID: "e4", Status: 400}, {ID: "e5", Status: 400},
	}}
	msg := e.Error()
	if !strings.Contains(msg, "rejected 5 of 7 events") || !strings.Contains(msg, "e3") || strings.Contains(msg, "e4") || !strings.Contains(msg, "and 2 more") {
		t.Errorf("msg = %q", msg)
	}
}

// The trace-level path: a rejection surfaces from RecordTrace and reaches
// OnFlushError, so the orchestrator's "trace not recorded" line names it.
func TestRecordTrace_SurfacesRejectedEvents(t *testing.T) {
	srv, _ := respondWith(t, http.StatusMultiStatus, partial207)
	var reported error
	_, err := RecordTrace(context.Background(), TraceRecord{},
		PublicKey("pk"), SecretKey("sk"), Host(srv.URL), OnFlushError(func(e error) { reported = e }))
	var rej *IngestionRejectedError
	if !errors.As(err, &rej) || !errors.As(reported, &rej) {
		t.Fatalf("err=%v reported=%v", err, reported)
	}
}
