// Package langfuse provides Langfuse tracing hooks for goai.
//
// Basic usage -- credentials come from env vars (LANGFUSE_PUBLIC_KEY, LANGFUSE_SECRET_KEY,
// LANGFUSE_HOST or LANGFUSE_BASE_URL):
//
//	result, err := goai.GenerateText(ctx, model,
//	    langfuse.WithTracing(),
//	    goai.WithPrompt("hello"),
//	)
//
// With options:
//
//	result, err := goai.GenerateText(ctx, model,
//	    langfuse.WithTracing(
//	        langfuse.TraceName("my-agent"),
//	        langfuse.UserID("user-123"),
//	        langfuse.Tags("prod", "v2"),
//	    ),
//	    goai.WithPrompt("hello"),
//	)
//
// Each call creates a fresh trace with isolated state. Concurrent calls are safe.
//
// Observation hierarchy:
//
//	Trace
//	└── Span("agent")            — wraps the entire run
//	    ├── Generation("step-1") — LLM call, child of agent span
//	    ├── Span("tool-name")    — tool execution, sibling of generation
//	    └── Generation("step-2") — final LLM call
package langfuse

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/zendev-sh/goai"
	"github.com/zendev-sh/goai/provider"
)

// TracingOption configures WithTracing.
type TracingOption func(*Config)

// TraceName sets the trace name (default "agent").
func TraceName(name string) TracingOption { return func(c *Config) { c.TraceName = name } }

// UserID sets the Langfuse user ID on the trace.
func UserID(id string) TracingOption { return func(c *Config) { c.UserID = id } }

// SessionID sets the Langfuse session ID for grouping traces.
func SessionID(id string) TracingOption { return func(c *Config) { c.SessionID = id } }

// Tags sets tags on the trace.
func Tags(tags ...string) TracingOption { return func(c *Config) { c.Tags = tags } }

// Metadata sets metadata on the trace.
func Metadata(m any) TracingOption { return func(c *Config) { c.Metadata = m } }

// Release sets the release identifier on the trace.
func Release(r string) TracingOption { return func(c *Config) { c.Release = r } }

// Version sets the version identifier on the trace.
func Version(v string) TracingOption { return func(c *Config) { c.Version = v } }

// Environment sets the environment (overrides LANGFUSE_ENV).
func Environment(env string) TracingOption { return func(c *Config) { c.Environment = env } }

// PromptName sets the Langfuse prompt name for linking generations.
func PromptName(name string) TracingOption { return func(c *Config) { c.PromptName = name } }

// PromptVersion sets the Langfuse prompt version for linking generations.
func PromptVersion(v int) TracingOption { return func(c *Config) { c.PromptVersion = v } }

// OnFlushError sets a callback for flush errors (default: silently discard).
func OnFlushError(fn func(error)) TracingOption { return func(c *Config) { c.OnFlushError = fn } }

// FlushTimeout bounds each HTTP flush to Langfuse (default DefaultFlushTimeout).
//
// Flushes run on a detached context — the run's own context is routinely
// cancelled or expired by the time the final flush happens, and tracing must
// still be delivered then — so this is the only thing bounding them. A flush
// that exceeds it fails (reported via OnFlushError) and its events are
// dropped, which is the documented best-effort contract; what it can no
// longer do is hold the caller. Agents running under a hard invocation cap
// (Lambda) size this against the time they hold back for the failure path.
func FlushTimeout(d time.Duration) TracingOption { return func(c *Config) { c.FlushTimeout = d } }

// DefaultFlushTimeout is the flush bound when FlushTimeout is not set. It is
// deliberately well under the client's 30s HTTP timeout: the end-of-run flush
// fires INSIDE goai's GenerateObject (from OnResponse on the error path), so
// every second it takes is a second the model phase appears to still be
// running — and on an invocation that has just overrun its budget, seconds it
// does not have (NEU-1416).
const DefaultFlushTimeout = 10 * time.Second

// maxUnparsedOutput caps the raw text recorded for a final step whose output
// is not valid JSON (a cut-off at the token ceiling).
const maxUnparsedOutput = 64 << 10

// PublicKey overrides the LANGFUSE_PUBLIC_KEY env var.
func PublicKey(key string) TracingOption { return func(c *Config) { c.PublicKey = key } }

// SecretKey overrides the LANGFUSE_SECRET_KEY env var.
func SecretKey(key string) TracingOption { return func(c *Config) { c.SecretKey = key } }

// Host overrides the LANGFUSE_HOST / LANGFUSE_BASE_URL env var.
func Host(host string) TracingOption { return func(c *Config) { c.Host = host } }

// WithTracing returns a goai.Option that enables Langfuse tracing for a single call.
// Credentials are read from env vars unless overridden via PublicKey/SecretKey/Host options.
// Each invocation creates a fresh trace -- safe for concurrent use.
//
// Each call allocates a small amount of state for trace isolation. The underlying
// HTTP connections are pooled by one package-level transport (see client.go),
// so creating a new http.Client per call does not waste TCP connections.
//
// If neither LANGFUSE_PUBLIC_KEY / LANGFUSE_SECRET_KEY env vars nor PublicKey/SecretKey
// options are set, WithTracing logs a warning to stderr and returns a no-op option.
func WithTracing(opts ...TracingOption) goai.Option {
	cfg := Config{}
	for _, o := range opts {
		o(&cfg)
	}

	// Resolve credentials from env vars if not set via options.
	pub := cfg.PublicKey
	if pub == "" {
		pub = os.Getenv("LANGFUSE_PUBLIC_KEY")
	}
	sec := cfg.SecretKey
	if sec == "" {
		sec = os.Getenv("LANGFUSE_SECRET_KEY")
	}
	if pub == "" || sec == "" {
		fmt.Fprintln(os.Stderr, "langfuse: warning: LANGFUSE_PUBLIC_KEY or LANGFUSE_SECRET_KEY not set, tracing disabled")
		return goai.WithOptions() // no-op
	}

	h := New(cfg)
	return goai.WithOptions(h.Run()...)
}

// Config configures Langfuse tracing. Credential fields override the corresponding
// env vars (LANGFUSE_PUBLIC_KEY, LANGFUSE_SECRET_KEY, LANGFUSE_HOST).
type Config struct {
	PublicKey string // overrides LANGFUSE_PUBLIC_KEY
	SecretKey string // overrides LANGFUSE_SECRET_KEY
	Host      string // overrides LANGFUSE_HOST / LANGFUSE_BASE_URL

	TraceName   string // defaults to "agent"
	UserID      string
	SessionID   string
	Tags        []string
	Metadata    any
	Release     string
	Version     string
	Environment string // falls back to LANGFUSE_ENV

	PromptName    string
	PromptVersion int

	// OnFlushError is called when the HTTP flush to Langfuse fails.
	// If nil, flush errors are silently discarded (tracing must not crash the app).
	OnFlushError func(error)

	// FlushTimeout bounds each HTTP flush; zero means DefaultFlushTimeout.
	FlushTimeout time.Duration
}

// flushBound is the effective per-flush bound.
func (c Config) flushBound() time.Duration {
	if c.FlushTimeout > 0 {
		return c.FlushTimeout
	}
	return DefaultFlushTimeout
}

// flushBounded runs one flush on a detached context bounded by flushBound and
// reports a failure through OnFlushError, naming the bound when the bound is
// what failed it.
func flushBounded(base context.Context, cfg Config, lc *client) {
	if base == nil {
		base = context.Background()
	}
	bound := cfg.flushBound()
	ctx, cancel := context.WithTimeout(base, bound)
	defer cancel()
	err := lc.flush(ctx)
	if err == nil || cfg.OnFlushError == nil {
		return
	}
	if errors.Is(err, context.DeadlineExceeded) {
		err = fmt.Errorf("langfuse: flush exceeded its %s bound, events dropped: %w", bound, err)
	}
	cfg.OnFlushError(err)
}

// Hooks holds the shared HTTP client and config.
// Create once; call Run() to get a fresh set of options per agent run.
//
// Deprecated: Use [WithTracing] instead.
type Hooks struct {
	cfg Config
	mu  sync.Mutex
	lc  *client // lazily initialised, shared across runs
}

// New returns a Hooks instance. The HTTP client is initialised lazily on the first Run() call.
//
// Deprecated: Use [WithTracing] instead.
func New(cfg Config) *Hooks {
	if cfg.TraceName == "" {
		cfg.TraceName = "agent"
	}
	if cfg.Environment == "" {
		cfg.Environment = os.Getenv("LANGFUSE_ENV")
	}
	return &Hooks{cfg: cfg}
}

// client returns the shared HTTP client, initialising it on first call.
func (h *Hooks) client() *client {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.lc == nil {
		pub := h.cfg.PublicKey
		if pub == "" {
			pub = os.Getenv("LANGFUSE_PUBLIC_KEY")
		}
		sec := h.cfg.SecretKey
		if sec == "" {
			sec = os.Getenv("LANGFUSE_SECRET_KEY")
		}
		host := h.cfg.Host
		if host == "" {
			host = os.Getenv("LANGFUSE_HOST")
		}
		if host == "" {
			host = os.Getenv("LANGFUSE_BASE_URL")
		}
		if host == "" {
			host = "https://cloud.langfuse.com"
		}
		h.lc = newClient(host, pub, sec)
	}
	return h.lc
}

// With returns a per-run options factory that includes langfuse tracing plus
// any additional options. Use this as the Options field on an agent:
//
//	Options: lf.With()                      // tracing only
//	Options: lf.With(goai.WithMaxSteps(5))  // tracing + extra options
//
// Deprecated: Use [WithTracing] instead.
func (h *Hooks) With(opts ...goai.Option) func() []goai.Option {
	return func() []goai.Option {
		return append(h.Run(), opts...)
	}
}

// Run returns a fresh set of goai options scoped to a single agent run.
// Safe to call concurrently — each call gets completely isolated state.
// OnToolCall may fire from parallel goroutines, so a mutex guards shared state.
//
// Deprecated: Use [WithTracing] instead.
func (h *Hooks) Run() []goai.Option {
	cfg := h.cfg
	lc := h.client()

	// Per-run state. Isolated per Run() call. Most hooks are called
	// sequentially by goai, but OnToolCall may fire from parallel
	// goroutines when multiple tool calls execute concurrently, so
	// mu guards writes to shared state.
	var (
		traceID     string
		agentSpanID string
		agentStart  time.Time
		traceInput  any
		// runCtx is the caller's context for the generation, captured from
		// the first OnRequest so the final flush in end() honours the
		// caller's cancellation/deadline instead of running detached.
		runCtx     context.Context
		gen        *pendingGen
		step       int
		lastObsEnd time.Time
		mu         sync.Mutex

		flushStop chan struct{}
		flushWG   sync.WaitGroup
	)

	// push enqueues an event and triggers an async flush for realtime delivery.
	// The ticker goroutine also flushes periodically to coalesce bursts.
	push := func(event ingestionEvent) {
		lc.appendEvents([]ingestionEvent{event})
	}

	// startFlusher spins up a background goroutine that flushes every 500ms.
	// Called lazily on the first OnRequest so runs with no LLM calls don't pay
	// for a ticker.
	startFlusher := func() {
		flushStop = make(chan struct{})
		flushWG.Add(1)
		go func() {
			defer flushWG.Done()
			ticker := time.NewTicker(500 * time.Millisecond)
			defer ticker.Stop()
			for {
				select {
				case <-ticker.C:
					flushBounded(context.Background(), cfg, lc)
				case <-flushStop:
					return
				}
			}
		}()
	}

	// end is declared before opts so WithOnStepFinish can reference it.
	var end func(result any)

	// OnToolCallStart is intentionally not registered here because
	// OnToolCall.StartTime already provides accurate timing for tool spans.
	opts := []goai.Option{
		goai.WithOnRequest(func(info goai.RequestInfo) {
			input := messagesToInput(info.Messages)

			if info.Ctx != nil {
				runCtx = info.Ctx
			}
			if traceID == "" {
				traceID = newID()
				agentSpanID = newID()
				agentStart = time.Now()
				traceInput = input
				startFlusher()

				// Emit the trace and agent span immediately so the run is
				// visible in Langfuse as soon as it starts. Both will be
				// re-emitted at end() with final output/endTime; Langfuse
				// treats repeat events with the same ID as upserts.
				traceMeta := cfg.Metadata
				if cfg.Environment != "" {
					traceMeta = mergeMeta(traceMeta, map[string]any{"environment": cfg.Environment})
				}
				push(ingestionEvent{
					ID:        newID(),
					Type:      eventTrace,
					Timestamp: formatTime(agentStart),
					Body: traceBody{
						ID:        traceID,
						Name:      cfg.TraceName,
						UserID:    cfg.UserID,
						SessionID: cfg.SessionID,
						Tags:      cfg.Tags,
						Metadata:  traceMeta,
						Release:   cfg.Release,
						Version:   cfg.Version,
						Input:     traceInput,
					},
				})
				push(ingestionEvent{
					ID:        newID(),
					Type:      eventSpan,
					Timestamp: formatTime(agentStart),
					Body: spanBody{
						ID:        agentSpanID,
						TraceID:   traceID,
						Name:      cfg.TraceName,
						StartTime: formatTime(agentStart),
						Input:     traceInput,
						Version:   cfg.Version,
					},
				})
			}

			// Finalise the previous generation with its tool-call output.
			if gen != nil {
				for i := len(info.Messages) - 1; i >= 0; i-- {
					if info.Messages[i].Role == provider.RoleAssistant {
						gen.output = lastAssistantOutput(info.Messages[i])
						break
					}
				}
				push(gen.toEvent(traceID, agentSpanID, cfg))
				gen = nil
			}

			step++

			// Offset the generation start time so it appears after preceding tool spans
			// in Langfuse's timeline. Langfuse sorts observations by startTime, so if
			// a generation starts at the same wall-clock instant as the last tool span
			// ends (common when tool execution is very fast), they render in wrong order.
			now := time.Now()
			if !lastObsEnd.IsZero() {
				if floor := lastObsEnd.Add(10 * time.Millisecond); !now.After(floor) {
					now = floor
				}
			}

			genMeta := map[string]any{}
			if cfg.Environment != "" {
				genMeta["environment"] = cfg.Environment
			}

			gen = &pendingGen{
				id:        newID(),
				name:      fmt.Sprintf("step-%d", step),
				model:     info.Model,
				startTime: now,
				input:     input,
				metadata:  genMeta,
			}
		}),

		goai.WithOnResponse(func(info goai.ResponseInfo) {
			if gen == nil {
				return
			}
			gen.latency = info.Latency
			if info.Error != nil {
				// Run failed (e.g. context cancelled, max retries exhausted).
				// Flush whatever partial trace we have so it's not silently lost.
				gen.level = levelError
				gen.statusMsg = info.Error.Error()
				end(nil)
				return
			}
			if info.FinishReason == provider.FinishLength {
				gen.level = levelWarning
			}
			if info.FinishReason != "" {
				gen.statusMsg = string(info.FinishReason)
			}
			gen.usage = usageBody{
				Input:  info.Usage.InputTokens,
				Output: info.Usage.OutputTokens,
				Total:  info.Usage.InputTokens + info.Usage.OutputTokens,
				Unit:   unitTokens,
			}
			if info.Usage.ReasoningTokens > 0 {
				gen.metadata["reasoning_tokens"] = info.Usage.ReasoningTokens
			}
			if info.Usage.CacheReadTokens > 0 {
				gen.metadata["cache_read_tokens"] = info.Usage.CacheReadTokens
			}
			if info.Usage.CacheWriteTokens > 0 {
				gen.metadata["cache_write_tokens"] = info.Usage.CacheWriteTokens
			}
		}),

		goai.WithOnStepFinish(func(step goai.StepResult) {
			// Emit reasoning summaries as a dedicated sibling span positioned
			// between the generation and its tool calls. This puts the
			// chain-of-thought at the top level of the trace view instead of
			// buried inside the generation's metadata.
			//
			// goai's non-streaming Responses path stores reasoning summaries in
			// step.ProviderMetadata["openai"]["reasoning"] as []{type,text}.
			if gen != nil && len(step.ProviderMetadata) > 0 {
				if reasoning := extractReasoning(step.ProviderMetadata); reasoning != "" {
					// Position the reasoning span at the end of the generation
					// so it sorts after the generation and before any tool
					// spans (which start at lastObsEnd + 10ms in OnToolCall).
					genEnd := gen.startTime.Add(gen.latency)
					mu.Lock()
					if genEnd.After(lastObsEnd) {
						lastObsEnd = genEnd
					}
					mu.Unlock()
					push(ingestionEvent{
						ID:        newID(),
						Type:      eventSpan,
						Timestamp: formatTime(genEnd),
						Body: spanBody{
							ID:                  newID(),
							TraceID:             traceID,
							ParentObservationID: agentSpanID,
							Name:                "reasoning",
							StartTime:           formatTime(genEnd),
							EndTime:             formatTime(genEnd),
							Output:              reasoning,
							Version:             cfg.Version,
							Metadata: map[string]any{
								"step":             step.Number,
								"reasoning_tokens": step.Usage.ReasoningTokens,
							},
						},
					})
				}
			}

			if step.FinishReason == provider.FinishToolCalls {
				return // intermediate step — more tool calls follow
			}
			var output any
			if step.Text != "" {
				if json.Unmarshal([]byte(step.Text), &output) != nil {
					// A step cut off at the output ceiling is not valid JSON;
					// dropping it left NEU-1534's truncated generations with a
					// null output. Keep the text, minus trailing whitespace (a
					// runaway can be 800 KB of it), capped.
					raw := strings.TrimRight(step.Text, " \t\r\n")
					if len(raw) > maxUnparsedOutput {
						raw = raw[:maxUnparsedOutput]
					}
					output = map[string]any{
						"unparsed_text":       raw,
						"text_chars":          len(step.Text),
						"trailing_whitespace": len(step.Text) - len(strings.TrimRight(step.Text, " \t\r\n")),
					}
				}
			}
			end(output)
		}),

		goai.WithOnToolCall(func(info goai.ToolCallInfo) {
			if traceID == "" {
				return
			}

			start := info.StartTime
			endTime := info.StartTime.Add(info.Duration)

			var inputVal any = info.Input
			if len(info.Input) > 0 {
				var parsed any
				if json.Unmarshal(info.Input, &parsed) == nil {
					inputVal = parsed
				}
			}
			outputVal := any(info.Output)
			if info.OutputObject != nil {
				outputVal = info.OutputObject
			}

			level := ""
			statusMsg := ""
			if info.Error != nil {
				level = levelError
				statusMsg = info.Error.Error()
			}

			mu.Lock()
			if endTime.After(lastObsEnd) {
				lastObsEnd = endTime
			}
			mu.Unlock()
			push(ingestionEvent{
				// ingestionEvent.ID is the deduplication key for the ingestion envelope;
				// spanBody.ID is the observation ID used for parent-child relationships.
				// Langfuse requires them to be distinct.
				ID:        newID(),
				Type:      eventSpan,
				Timestamp: formatTime(start),
				Body: spanBody{
					ID:                  newID(),
					TraceID:             traceID,
					ParentObservationID: agentSpanID,
					Name:                info.ToolName,
					StartTime:           formatTime(start),
					EndTime:             formatTime(endTime),
					Input:               inputVal,
					Output:              outputVal,
					Version:             cfg.Version,
					Metadata: map[string]any{
						"tool_call_id": info.ToolCallID,
						"step":         info.Step,
					},
					Level:         level,
					StatusMessage: statusMsg,
				},
			})
		}),
	}

	end = func(result any) {
		if traceID == "" {
			return
		}

		if gen != nil {
			gen.output = result
			push(gen.toEvent(traceID, agentSpanID, cfg))
			gen = nil
		}

		now := time.Now()

		traceMeta := cfg.Metadata
		if cfg.Environment != "" {
			traceMeta = mergeMeta(traceMeta, map[string]any{"environment": cfg.Environment})
		}

		// Re-emit the trace and agent span with final output/endTime.
		// Langfuse treats repeat events with the same ID as upserts.
		push(ingestionEvent{
			ID:        newID(),
			Type:      eventTrace,
			Timestamp: formatTime(now),
			Body: traceBody{
				ID:        traceID,
				Name:      cfg.TraceName,
				UserID:    cfg.UserID,
				SessionID: cfg.SessionID,
				Tags:      cfg.Tags,
				Metadata:  traceMeta,
				Release:   cfg.Release,
				Version:   cfg.Version,
				Input:     traceInput,
				Output:    result,
			},
		})
		push(ingestionEvent{
			ID:        newID(),
			Type:      eventSpan,
			Timestamp: formatTime(now),
			Body: spanBody{
				ID:        agentSpanID,
				TraceID:   traceID,
				Name:      cfg.TraceName,
				StartTime: formatTime(agentStart),
				EndTime:   formatTime(now),
				Input:     traceInput,
				Output:    result,
				Version:   cfg.Version,
			},
		})

		// Clear run state — the run is complete.
		traceID = ""
		agentSpanID = ""

		// Stop the background flusher and do a final synchronous flush so
		// the caller can be confident everything reached Langfuse before
		// the process exits. Both are bounded by cfg.FlushTimeout: this runs
		// inside goai's GenerateObject (OnResponse fires it on the error
		// path), so an unbounded flush against a slow or unreachable Langfuse
		// held the model phase open for up to the client's 30s timeout — on a
		// Lambda that had just overrun its budget, that was the difference
		// between failing loudly and being killed at the cap (NEU-1416).
		// Worst case here is 2 × FlushTimeout: a ticker flush in flight plus
		// the final one.
		if flushStop != nil {
			close(flushStop)
			flushStop = nil
			flushWG.Wait()
		}
		// The flush runs under the caller's context while it is alive. Once
		// the run has been cancelled (the error path above lands here with a
		// dead ctx) the partial trace is still worth keeping, so fall back to
		// a detached context that inherits the values but not the
		// cancellation. Either way flushBounded caps it at cfg.FlushTimeout.
		base := context.Background()
		if runCtx != nil {
			if runCtx.Err() == nil {
				base = runCtx
			} else {
				base = context.WithoutCancel(runCtx)
			}
		}
		flushBounded(base, cfg, lc)
	}

	return opts
}

// --- Langfuse API body types -----------------------------------------------

type traceBody struct {
	ID        string   `json:"id"`
	Name      string   `json:"name,omitempty"`
	UserID    string   `json:"userId,omitempty"`
	SessionID string   `json:"sessionId,omitempty"`
	Tags      []string `json:"tags,omitempty"`
	Metadata  any      `json:"metadata,omitempty"`
	Release   string   `json:"release,omitempty"`
	Version   string   `json:"version,omitempty"`
	Input     any      `json:"input,omitempty"`
	Output    any      `json:"output,omitempty"`
}

type spanBody struct {
	ID                  string `json:"id"`
	TraceID             string `json:"traceId"`
	ParentObservationID string `json:"parentObservationId,omitempty"`
	Name                string `json:"name,omitempty"`
	StartTime           string `json:"startTime,omitempty"`
	EndTime             string `json:"endTime,omitempty"`
	Input               any    `json:"input,omitempty"`
	Output              any    `json:"output,omitempty"`
	Version             string `json:"version,omitempty"`
	Metadata            any    `json:"metadata,omitempty"`
	Level               string `json:"level,omitempty"`
	StatusMessage       string `json:"statusMessage,omitempty"`
}

type generationBody struct {
	ID                  string     `json:"id"`
	TraceID             string     `json:"traceId"`
	ParentObservationID string     `json:"parentObservationId,omitempty"`
	Name                string     `json:"name,omitempty"`
	StartTime           string     `json:"startTime,omitempty"`
	EndTime             string     `json:"endTime,omitempty"`
	Input               any        `json:"input,omitempty"`
	Output              any        `json:"output,omitempty"`
	Version             string     `json:"version,omitempty"`
	Metadata            any        `json:"metadata,omitempty"`
	Level               string     `json:"level,omitempty"`
	StatusMessage       string     `json:"statusMessage,omitempty"`
	Model               string     `json:"model,omitempty"`
	Usage               *usageBody `json:"usage,omitempty"`
	PromptName          string     `json:"promptName,omitempty"`
	PromptVersion       int        `json:"promptVersion,omitempty"`
}

type usageBody struct {
	Input  int    `json:"input,omitempty"`
	Output int    `json:"output,omitempty"`
	Total  int    `json:"total,omitempty"`
	Unit   string `json:"unit,omitempty"`
}

const (
	eventTrace      = "trace-create"
	eventSpan       = "span-create"
	eventGeneration = "generation-create"
	levelWarning    = "WARNING"
	levelError      = "ERROR"
	unitTokens      = "TOKENS"
)

// pendingGen accumulates data for a single LLM generation step across
// OnRequest → OnResponse → lazy finalisation.
type pendingGen struct {
	id        string
	name      string
	model     string
	startTime time.Time
	input     any
	metadata  map[string]any

	// populated by OnResponse
	latency   time.Duration
	level     string
	statusMsg string
	usage     usageBody

	// populated lazily (next OnRequest or end)
	output any
}

func (g *pendingGen) toEvent(traceID, parentID string, cfg Config) ingestionEvent {
	endTime := g.startTime.Add(g.latency)
	body := generationBody{
		ID:                  g.id,
		TraceID:             traceID,
		ParentObservationID: parentID,
		Name:                g.name,
		Model:               g.model,
		StartTime:           formatTime(g.startTime),
		EndTime:             formatTime(endTime),
		Input:               g.input,
		Output:              g.output,
		Version:             cfg.Version,
		PromptName:          cfg.PromptName,
		PromptVersion:       cfg.PromptVersion,
		Metadata:            g.metadata,
		Level:               g.level,
		StatusMessage:       g.statusMsg,
	}
	if g.usage.Input > 0 || g.usage.Output > 0 || g.usage.Total > 0 {
		body.Usage = &g.usage
	}
	return ingestionEvent{
		ID:        newID(),
		Type:      eventGeneration,
		Timestamp: formatTime(g.startTime),
		Body:      body,
	}
}

func mergeMeta(base any, extra map[string]any) map[string]any {
	result := make(map[string]any, len(extra))
	if m, ok := base.(map[string]any); ok {
		for k, v := range m {
			result[k] = v
		}
	}
	for k, v := range extra {
		result[k] = v
	}
	return result
}

func toolCallMap(p provider.Part) map[string]any {
	tc := map[string]any{"id": p.ToolCallID, "name": p.ToolName}
	if len(p.ToolInput) > 0 {
		var parsed any
		if json.Unmarshal(p.ToolInput, &parsed) == nil {
			tc["input"] = parsed
		} else {
			tc["input"] = string(p.ToolInput)
		}
	}
	return tc
}

// extractReasoning pulls reasoning-summary text out of a provider metadata map
// and returns it as a single concatenated string. Supported shapes:
//
//	providerMetadata["openai"]["reasoning"]  = []{type, text}  (OpenAI Responses)
//	providerMetadata["google"]["reasoning"]  = "..."           (Gemini thinking)
//	providerMetadata["anthropic"]["thinking"] = "..."          (Claude thinking)
//
// Returns "" when no reasoning is present.
func extractReasoning(pm map[string]map[string]any) string {
	var parts []string
	for _, meta := range pm {
		for _, key := range []string{"reasoning", "thinking"} {
			v, ok := meta[key]
			if !ok {
				continue
			}
			switch vv := v.(type) {
			case string:
				if vv != "" {
					parts = append(parts, vv)
				}
			case []map[string]any:
				for _, entry := range vv {
					if s, ok := entry["text"].(string); ok && s != "" {
						parts = append(parts, s)
					}
				}
			case []any:
				for _, entry := range vv {
					if m, ok := entry.(map[string]any); ok {
						if s, ok := m["text"].(string); ok && s != "" {
							parts = append(parts, s)
						}
					} else if s, ok := entry.(string); ok && s != "" {
						parts = append(parts, s)
					}
				}
			}
		}
	}
	if len(parts) == 0 {
		return ""
	}
	return strings.Join(parts, "\n\n")
}

func messagesToInput(msgs []provider.Message) []map[string]any {
	result := make([]map[string]any, 0, len(msgs))
	for _, m := range msgs {
		entry := map[string]any{"role": string(m.Role)}
		var textParts []string
		var toolCalls []map[string]any
		for _, p := range m.Content {
			switch p.Type {
			case provider.PartText:
				if p.Text != "" {
					textParts = append(textParts, p.Text)
				}
			case provider.PartToolCall:
				toolCalls = append(toolCalls, toolCallMap(p))
			case provider.PartToolResult:
				entry["tool_call_id"] = p.ToolCallID
				if p.ToolOutput != "" {
					var parsed any
					if json.Unmarshal([]byte(p.ToolOutput), &parsed) == nil {
						entry["content"] = parsed
					} else {
						entry["content"] = p.ToolOutput
					}
				}
			}
		}
		if len(toolCalls) > 0 {
			entry["tool_calls"] = toolCalls
		} else if len(textParts) > 0 {
			if _, set := entry["content"]; !set {
				if len(textParts) == 1 {
					entry["content"] = textParts[0]
				} else {
					entry["content"] = textParts
				}
			}
		}
		result = append(result, entry)
	}
	return result
}

func lastAssistantOutput(m provider.Message) any {
	var toolCalls []map[string]any
	var textParts []string
	for _, p := range m.Content {
		switch p.Type {
		case provider.PartToolCall:
			toolCalls = append(toolCalls, toolCallMap(p))
		case provider.PartText:
			if p.Text != "" {
				textParts = append(textParts, p.Text)
			}
		}
	}
	if len(toolCalls) > 0 {
		return toolCalls
	}
	if len(textParts) == 1 {
		return textParts[0]
	}
	return textParts
}
