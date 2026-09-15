package langfuse

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"syscall"
	"time"
)

// idleConnTimeout caps how long a keep-alive connection to Langfuse may sit
// idle in our pool before we drop it ourselves. Langfuse runs as a single
// Node process whose server closes idle keep-alive sockets after
// keepAliveTimeout (5 s by default). Our final flush fires after the whole
// LLM phase, so with Go's default IdleConnTimeout (90 s) the pooled
// connection was always one the server had already closed: the POST went
// out on a dead socket and came back as `read: connection reset by peer`
// or EOF, and the trace was lost (NEU-1375). Staying under the server's
// timeout means any connection we reuse is one the server still holds.
// Keep-alives stay on because the 500 ms ticker flushes during a burst do
// benefit from a warm connection; DisableKeepAlives would pay a TLS
// handshake per flush for no gain.
const idleConnTimeout = 3 * time.Second

// transport is shared by every client in the process, so connections are
// pooled across WithTracing calls the way http.DefaultTransport pooled them
// before the per-package tuning above.
var transport = newTransport()

func newTransport() *http.Transport {
	t, ok := http.DefaultTransport.(*http.Transport)
	if !ok {
		t = &http.Transport{Proxy: http.ProxyFromEnvironment}
	} else {
		t = t.Clone()
	}
	t.IdleConnTimeout = idleConnTimeout
	return t
}

// client accumulates Langfuse ingestion events and sends them in a single batch on flush.
type client struct {
	host       string
	auth       string // base64(publicKey:secretKey)
	httpClient *http.Client

	mu     sync.Mutex
	events []ingestionEvent
}

type ingestionEvent struct {
	ID        string `json:"id"`
	Type      string `json:"type"`
	Timestamp string `json:"timestamp"`
	Body      any    `json:"body"`
}

func newClient(host, publicKey, secretKey string) *client {
	return &client{
		host:       strings.TrimRight(host, "/"),
		auth:       base64.StdEncoding.EncodeToString([]byte(publicKey + ":" + secretKey)),
		httpClient: &http.Client{Timeout: 30 * time.Second, Transport: transport},
	}
}

// appendEvents adds pre-built events to the batch in a single lock operation.
func (c *client) appendEvents(events []ingestionEvent) {
	c.mu.Lock()
	c.events = append(c.events, events...)
	c.mu.Unlock()
}

// flush sends all buffered events to Langfuse in a single POST and clears the queue.
// Events are cleared before the POST to avoid double-sending across flushes.
//
// A POST that dies on the wire (connection reset, EOF: the server dropped
// the socket under us) is retried exactly once on a fresh connection. That
// is safe because Langfuse deduplicates on the per-event `id` in the
// ingestion envelope ("We use the event id within this envelope to
// deduplicate messages to avoid processing the same event twice"), so a
// batch that was in fact processed before the socket died is a no-op the
// second time. HTTP-level failures (4xx/5xx) and context cancellation are
// not retried. If the retry fails too, those events are permanently lost -
// this is intentional: observability is best-effort and must never block or
// retry indefinitely.
func (c *client) flush(ctx context.Context) error {
	c.mu.Lock()
	if len(c.events) == 0 {
		c.mu.Unlock()
		return nil
	}
	events := c.events
	c.events = nil
	c.mu.Unlock()

	payload, err := json.Marshal(map[string]any{"batch": events})
	if err != nil {
		return fmt.Errorf("langfuse: marshal batch: %w", err)
	}

	err = c.post(ctx, payload)
	if err != nil && ctx.Err() == nil && isConnectionDropped(err) {
		err = c.post(ctx, payload)
	}
	return err
}

// post sends one ingestion batch. The request is rebuilt per attempt so the
// body reader starts at offset 0 on a retry.
func (c *client) post(ctx context.Context, payload []byte) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		c.host+"/api/public/ingestion", bytes.NewReader(payload))
	if err != nil {
		return fmt.Errorf("langfuse: create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Basic "+c.auth)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("langfuse: send batch: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	body, _ := io.ReadAll(io.LimitReader(resp.Body, maxResponseBytes))
	_, _ = io.Copy(io.Discard, resp.Body)

	if resp.StatusCode >= 400 {
		if s := snippet(string(body), 300); s != "" {
			return fmt.Errorf("langfuse: ingestion failed with status %d: %s", resp.StatusCode, s)
		}
		return fmt.Errorf("langfuse: ingestion failed with status %d", resp.StatusCode)
	}
	return rejectedEvents(body)
}

// maxResponseBytes caps how much of an ingestion response is read. A 207 body
// lists every event with its outcome, so it grows with the batch; the errors
// worth reporting fit well inside this.
const maxResponseBytes = 1 << 20

// IngestionRejectedError reports events Langfuse refused inside a response
// whose HTTP status said the batch was accepted.
//
// The ingestion endpoint answers 207 Multi-Status for every batch and puts the
// per-event verdict in the body: {"successes":[...],"errors":[...]}. Treating
// "status < 400" as success — as this client did until 2026-09-15 — drops a
// rejected event (a malformed timestamp, a wrong type, an oversized body)
// without a trace anywhere: no error, no OnFlushError, no log line. Verified
// against Langfuse 3.225.1: a batch of one valid and one invalid event
// returned HTTP 207 with the invalid one under "errors" and status 400.
//
// It is not retried: a rejected event is rejected for its content, and
// resending it would be rejected again.
type IngestionRejectedError struct {
	Accepted int
	Rejected []RejectedEvent
}

// RejectedEvent is one entry of a 207 response's "errors" list.
type RejectedEvent struct {
	ID      string `json:"id"`
	Status  int    `json:"status"`
	Message string `json:"message"`
	Error   any    `json:"error"`
}

func (e *IngestionRejectedError) Error() string {
	var b strings.Builder
	fmt.Fprintf(&b, "langfuse: ingestion rejected %d of %d events", len(e.Rejected), len(e.Rejected)+e.Accepted)
	for i, r := range e.Rejected {
		if i == 3 {
			fmt.Fprintf(&b, "; and %d more", len(e.Rejected)-i)
			break
		}
		fmt.Fprintf(&b, "; %s: status %d %s", r.ID, r.Status, r.Message)
		if detail := errorDetail(r.Error); detail != "" {
			fmt.Fprintf(&b, " (%s)", snippet(detail, 200))
		}
	}
	return b.String()
}

// rejectedEvents returns an *IngestionRejectedError when an accepted response
// still lists rejected events, and nil otherwise — including for a body that
// is empty or not the documented shape, which says nothing about rejection.
func rejectedEvents(body []byte) error {
	if len(bytes.TrimSpace(body)) == 0 {
		return nil
	}
	var parsed struct {
		Successes []json.RawMessage `json:"successes"`
		Errors    []RejectedEvent   `json:"errors"`
	}
	if err := json.Unmarshal(body, &parsed); err != nil || len(parsed.Errors) == 0 {
		return nil
	}
	return &IngestionRejectedError{Accepted: len(parsed.Successes), Rejected: parsed.Errors}
}

func errorDetail(v any) string {
	switch d := v.(type) {
	case nil:
		return ""
	case string:
		return strings.Join(strings.Fields(d), " ")
	default:
		b, err := json.Marshal(d)
		if err != nil {
			return ""
		}
		return string(b)
	}
}

// snippet collapses whitespace and caps s at max bytes on a rune boundary.
func snippet(s string, max int) string {
	s = strings.Join(strings.Fields(s), " ")
	if len(s) <= max {
		return s
	}
	cut := max
	for cut > 0 && s[cut]&0xC0 == 0x80 {
		cut--
	}
	return s[:cut] + "…"
}

// isConnectionDropped reports whether err is the "peer closed the socket"
// class of transport failure: the request never produced a response
// because the connection was reset or hit EOF. Anything else (DNS, TLS,
// timeouts, HTTP status errors) is not retried.
func isConnectionDropped(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, io.EOF) || errors.Is(err, io.ErrUnexpectedEOF) ||
		errors.Is(err, syscall.ECONNRESET) || errors.Is(err, syscall.EPIPE) {
		return true
	}
	// net/http keeps this sentinel unexported; it is what a reused
	// connection the server already closed surfaces as for a POST.
	return strings.Contains(err.Error(), "server closed idle connection")
}

// newID returns a random UUID v4 string using only the standard library.
func newID() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	b[6] = (b[6] & 0x0f) | 0x40 // version 4
	b[8] = (b[8] & 0x3f) | 0x80 // variant bits
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:])
}

// formatTime formats a time.Time as ISO 8601 with millisecond precision for Langfuse.
// Returns an empty string for zero values (used with omitempty).
func formatTime(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.UTC().Format("2006-01-02T15:04:05.000Z07:00")
}
