package wt

import (
	"context"
	"errors"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"
)

// Client reports errors to a WatchTower instance. Use Init to configure the
// process-wide client, or construct one directly with NewClient for tests.
type Client struct {
	project  string
	tag      string
	release  string
	sample   float64
	disabled bool

	tr    *transport
	batch *batch
	log   *log.Logger
}

// Config configures the SDK. BaseURL, APIKey and Project are required.
type Config struct {
	// BaseURL is the WatchTower instance, e.g. "http://localhost:8080".
	BaseURL string
	// APIKey authenticates every report.
	APIKey string
	// Project labels every report from this service.
	Project string
	// Tag is the default tag applied to every report (overridable per report).
	Tag string
	// Release is an optional version/git commit attached to every report.
	Release string
	// SampleRate in [0,1]; 1.0 reports everything, 0.0 reports nothing.
	SampleRate float64
	// BatchInterval controls how often buffered events are flushed.
	BatchInterval time.Duration
	// BatchSize flushes as soon as this many events are buffered.
	BatchSize int
	// MaxQueueSize bounds the buffer; events beyond it are dropped.
	MaxQueueSize int
	// HTTPTimeout bounds each ingestion request.
	HTTPTimeout time.Duration
	// HTTPClient overrides the default client (useful for tests).
	HTTPClient *http.Client
	// Logger receives SDK diagnostics; defaults to stderr.
	Logger *log.Logger
	// Disable turns the SDK into a no-op (local development).
	Disable bool
}

// DefaultConfig returns the recommended configuration.
func DefaultConfig() Config {
	return Config{
		SampleRate:    1.0,
		BatchInterval: 5 * time.Second,
		BatchSize:     100,
		MaxQueueSize:  5000,
		HTTPTimeout:   10 * time.Second,
	}
}

// ConfigFromEnv builds a Config from WATCHTOWER_* environment variables,
// falling back to DefaultConfig for unset values.
func ConfigFromEnv() Config {
	cfg := DefaultConfig()
	cfg.BaseURL = os.Getenv("WATCHTOWER_BASE_URL")
	cfg.APIKey = os.Getenv("WATCHTOWER_API_KEY")
	cfg.Project = os.Getenv("WATCHTOWER_PROJECT")
	cfg.Tag = os.Getenv("WATCHTOWER_TAG")
	cfg.Release = os.Getenv("WATCHTOWER_RELEASE")
	if v := os.Getenv("WATCHTOWER_SAMPLE_RATE"); v != "" {
		if f, err := strconv.ParseFloat(v, 64); err == nil {
			cfg.SampleRate = f
		}
	}
	return cfg
}

// NewClient builds and starts a client. The returned client must be closed via
// Close to flush buffered events on shutdown.
func NewClient(cfg Config) (*Client, error) {
	if cfg.Logger == nil {
		cfg.Logger = log.New(os.Stderr, "", log.LstdFlags)
	}

	c := &Client{
		project:  cfg.Project,
		tag:      cfg.Tag,
		release:  cfg.Release,
		disabled: cfg.Disable,
		log:      cfg.Logger,
	}
	if cfg.Disable {
		return c, nil
	}

	if strings.TrimSpace(cfg.BaseURL) == "" {
		return nil, errors.New("watchtower: BaseURL is required")
	}
	if strings.TrimSpace(cfg.APIKey) == "" {
		return nil, errors.New("watchtower: APIKey is required")
	}
	if strings.TrimSpace(cfg.Project) == "" {
		return nil, errors.New("watchtower: Project is required")
	}

	if cfg.SampleRate < 0 {
		cfg.SampleRate = 0
	}
	if cfg.SampleRate > 1 {
		cfg.SampleRate = 1
	}
	if cfg.BatchInterval <= 0 {
		cfg.BatchInterval = 5 * time.Second
	}
	if cfg.BatchSize <= 0 {
		cfg.BatchSize = 100
	}
	if cfg.MaxQueueSize <= 0 {
		cfg.MaxQueueSize = 5000
	}
	if cfg.HTTPTimeout <= 0 {
		cfg.HTTPTimeout = 10 * time.Second
	}

	c.sample = cfg.SampleRate
	c.tr = newTransport(cfg)
	c.batch = newBatch(c.tr, cfg.BatchInterval, cfg.BatchSize, cfg.MaxQueueSize, cfg.Logger)
	return c, nil
}

// Report captures err and enqueues it for delivery. It never blocks and never
// panics; reporting is asynchronous. A nil error is a no-op.
func (c *Client) Report(err error, opts ...ReportOption) {
	if !c.reportable(err) {
		return
	}
	e := c.newEvent(err, opts...)
	if !c.batch.enqueue(e) {
		c.log.Printf("watchtower: dropped event: %s", e.Message)
	}
}

// ReportSync reports err immediately over HTTP and returns the backend result.
// It is intended for tests, tooling and low-volume critical paths.
func (c *Client) ReportSync(err error, opts ...ReportOption) ([]Result, error) {
	if !c.reportable(err) {
		return nil, nil
	}
	e := c.newEvent(err, opts...)
	ctx, cancel := context.WithTimeout(context.Background(), c.tr.client.Timeout)
	defer cancel()
	return c.tr.send(ctx, []event{e})
}

// reportPanic reports a recovered panic value, using the panic stack captured
// at recovery time.
func (c *Client) reportPanic(v any, opts ...ReportOption) {
	if c == nil || c.disabled {
		return
	}
	if c.sample < 1 && rand.Float64() >= c.sample {
		return
	}
	err, ok := v.(error)
	if !ok {
		err = fmt.Errorf("%v", v)
	}
	e := event{
		Message:    err.Error(),
		Project:    c.project,
		Tag:        c.tag,
		StackTrace: capturePanicStack(),
	}
	for _, o := range opts {
		o(&e)
	}
	enrich(&e, c.release)
	_ = c.batch.enqueue(e)
}

// reportable applies the nil check, disabled flag and sampling gate.
func (c *Client) reportable(err error) bool {
	if err == nil || c == nil || c.disabled || c.batch == nil {
		return false
	}
	if c.sample < 1 && rand.Float64() >= c.sample {
		return false
	}
	return true
}

// newEvent builds an event from err, applies options, then enrichment.
func (c *Client) newEvent(err error, opts ...ReportOption) event {
	e := event{
		Message:    err.Error(),
		Project:    c.project,
		Tag:        c.tag,
		StackTrace: captureStack(),
	}
	for _, o := range opts {
		o(&e)
	}
	enrich(&e, c.release)
	return e
}

// Flush synchronously delivers any buffered events.
func (c *Client) Flush() {
	if c != nil && c.batch != nil {
		c.batch.Flush()
	}
}

// Close flushes buffered events and stops the flusher goroutine. After Close,
// further reports are dropped. Safe to call multiple times.
func (c *Client) Close() {
	if c != nil && c.batch != nil {
		c.batch.Close()
	}
}

// ---------------------------------------------------------------------------
// Process-wide default client (docs-style wt.Init / wt.Report).
// ---------------------------------------------------------------------------

var (
	globalMu sync.RWMutex
	global   *Client
)

// Init configures the process-wide client. Calling Init more than once is a
// no-op; use NewClient to manage multiple clients.
func Init(cfg Config) error {
	c, err := NewClient(cfg)
	if err != nil {
		return err
	}
	globalMu.Lock()
	if global == nil {
		global = c
	}
	globalMu.Unlock()
	return nil
}

func defaultClient() *Client {
	globalMu.RLock()
	defer globalMu.RUnlock()
	return global
}

// Report sends err via the process-wide client (see Init).
func Report(err error, opts ...ReportOption) {
	if c := defaultClient(); c != nil {
		c.Report(err, opts...)
	}
}

// ReportSync sends err immediately via the process-wide client (see Init).
func ReportSync(err error, opts ...ReportOption) ([]Result, error) {
	if c := defaultClient(); c != nil {
		return c.ReportSync(err, opts...)
	}
	return nil, nil
}

// Flush delivers any buffered events via the process-wide client.
func Flush() {
	if c := defaultClient(); c != nil {
		c.Flush()
	}
}

// Close flushes and shuts down the process-wide client.
func Close() {
	if c := defaultClient(); c != nil {
		c.Close()
	}
}

// ---------------------------------------------------------------------------
// Report options.
// ---------------------------------------------------------------------------

// ReportOption mutates a reported event.
type ReportOption func(*event)

// WithTag sets the tag for a single report.
func WithTag(tag string) ReportOption {
	return func(e *event) { e.Tag = tag }
}

// WithContext merges key/value context into a single report.
func WithContext(ctx map[string]any) ReportOption {
	return func(e *event) {
		if len(ctx) == 0 {
			return
		}
		if e.Context == nil {
			e.Context = map[string]any{}
		}
		for k, v := range ctx {
			e.Context[k] = v
		}
	}
}

// WithTimestamp overrides the event timestamp.
func WithTimestamp(ts time.Time) ReportOption {
	return func(e *event) { e.Timestamp = &ts }
}

// WithErrorType overrides the derived error type.
func WithErrorType(t string) ReportOption {
	return func(e *event) { e.ErrorType = t }
}
