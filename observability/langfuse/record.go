package langfuse

import (
	"context"
	"errors"
	"fmt"
	"os"
	"time"
)

// TraceRecord is a complete run that made no model call — a decision taken in
// code — recorded after the fact so it sits in Langfuse next to the model runs
// it replaces, under the same trace name, session and tags.
type TraceRecord struct {
	Start, End    time.Time
	Input, Output any
	// Level and StatusMessage mark the root span ("WARNING", "ERROR").
	Level         string
	StatusMessage string
	// Spans are children of the root span, in order.
	Spans []SpanRecord
}

// SpanRecord is one child span of a TraceRecord.
type SpanRecord struct {
	Name          string
	Start, End    time.Time
	Input, Output any
	Metadata      any
	Level         string
	StatusMessage string
}

// RecordTrace sends rec as one trace — the trace, a root span named after the
// trace, and rec.Spans beneath it — in a single synchronous flush bounded by
// FlushTimeout. It returns the trace id.
//
// The shape matches what WithTracing produces for a model run (trace + root
// span of the same name), so queries over traces and observations treat both
// alike; there is simply no generation.
//
// Without credentials it records nothing and returns ("", nil): tracing is
// optional and a CLI or test run must not fail for lack of it. A failed flush
// is returned (and reported through OnFlushError) rather than retried —
// observability is best-effort.
func RecordTrace(ctx context.Context, rec TraceRecord, opts ...TracingOption) (string, error) {
	cfg := Config{}
	for _, o := range opts {
		o(&cfg)
	}
	h := New(cfg)
	cfg = h.cfg
	if (cfg.PublicKey == "" && os.Getenv("LANGFUSE_PUBLIC_KEY") == "") ||
		(cfg.SecretKey == "" && os.Getenv("LANGFUSE_SECRET_KEY") == "") {
		return "", nil
	}
	lc := h.client()

	start, end := rec.Start, rec.End
	if start.IsZero() {
		start = time.Now()
	}
	if end.Before(start) {
		end = start
	}

	traceID, rootID := newID(), newID()
	meta := cfg.Metadata
	if cfg.Environment != "" {
		meta = mergeMeta(meta, map[string]any{"environment": cfg.Environment})
	}
	events := []ingestionEvent{
		{
			ID:        newID(),
			Type:      eventTrace,
			Timestamp: formatTime(start),
			Body: traceBody{
				ID:        traceID,
				Name:      cfg.TraceName,
				UserID:    cfg.UserID,
				SessionID: cfg.SessionID,
				Tags:      cfg.Tags,
				Metadata:  meta,
				Release:   cfg.Release,
				Version:   cfg.Version,
				Input:     rec.Input,
				Output:    rec.Output,
			},
		},
		{
			ID:        newID(),
			Type:      eventSpan,
			Timestamp: formatTime(start),
			Body: spanBody{
				ID:            rootID,
				TraceID:       traceID,
				Name:          cfg.TraceName,
				StartTime:     formatTime(start),
				EndTime:       formatTime(end),
				Input:         rec.Input,
				Output:        rec.Output,
				Version:       cfg.Version,
				Metadata:      meta,
				Level:         rec.Level,
				StatusMessage: rec.StatusMessage,
			},
		},
	}
	for _, s := range rec.Spans {
		ss, se := s.Start, s.End
		if ss.IsZero() {
			ss = start
		}
		if se.Before(ss) {
			se = ss
		}
		events = append(events, ingestionEvent{
			ID:        newID(),
			Type:      eventSpan,
			Timestamp: formatTime(ss),
			Body: spanBody{
				ID:                  newID(),
				TraceID:             traceID,
				ParentObservationID: rootID,
				Name:                s.Name,
				StartTime:           formatTime(ss),
				EndTime:             formatTime(se),
				Input:               s.Input,
				Output:              s.Output,
				Version:             cfg.Version,
				Metadata:            s.Metadata,
				Level:               s.Level,
				StatusMessage:       s.StatusMessage,
			},
		})
	}
	lc.appendEvents(events)

	base := context.Background()
	if ctx != nil {
		base = context.WithoutCancel(ctx)
	}
	bound := cfg.flushBound()
	fctx, cancel := context.WithTimeout(base, bound)
	defer cancel()
	if err := lc.flush(fctx); err != nil {
		if errors.Is(err, context.DeadlineExceeded) {
			err = fmt.Errorf("langfuse: flush exceeded its %s bound, events dropped: %w", bound, err)
		}
		if cfg.OnFlushError != nil {
			cfg.OnFlushError(err)
		}
		return traceID, err
	}
	return traceID, nil
}
