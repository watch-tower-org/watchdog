package wt

import (
	"context"
	"errors"
	"io"
	"log"
	"sync/atomic"
	"testing"
	"time"
)

type sentinelErr struct{ msg string }

func (e *sentinelErr) Error() string { return e.msg }

func discardLogger() *log.Logger { return log.New(io.Discard, "", 0) }

func TestErrorTypeDerivedFromError(t *testing.T) {
	c := newCapture(t, "wt_secret")
	cl, _ := NewClient(testConfig(c, "wt_secret"))
	defer cl.Close()

	cl.Report(&sentinelErr{msg: "boom"})
	cl.Flush()
	reqs := c.requests()
	if len(reqs) != 1 || len(reqs[0].Events) != 1 {
		t.Fatalf("expected 1 event, got %d requests / %d events", len(reqs), len(reqs[0].Events))
	}
	if got := reqs[0].Events[0].ErrorType; got != "*wt.sentinelErr" {
		t.Errorf("error_type = %q, want *wt.sentinelErr", got)
	}
}

func TestErrorTypeOverriddenByOption(t *testing.T) {
	c := newCapture(t, "wt_secret")
	cl, _ := NewClient(testConfig(c, "wt_secret"))
	defer cl.Close()

	cl.Report(errors.New("boom"), WithErrorType("db.timeout"))
	cl.Flush()
	reqs := c.requests()
	if got := reqs[0].Events[0].ErrorType; got != "db.timeout" {
		t.Errorf("error_type = %q, want db.timeout", got)
	}
}

func TestReportPanicSetsErrorType(t *testing.T) {
	c := newCapture(t, "wt_secret")
	cl, _ := NewClient(testConfig(c, "wt_secret"))
	defer cl.Close()

	cl.ReportPanic(&sentinelErr{msg: "panicked"})
	cl.Flush()
	reqs := c.requests()
	if got := reqs[0].Events[0].ErrorType; got != "*wt.sentinelErr" {
		t.Errorf("error_type = %q, want *wt.sentinelErr", got)
	}
}

func TestGlobalReportPanicUsesProcessWideClient(t *testing.T) {
	c := newCapture(t, "wt_secret")
	if err := Init(testConfig(c, "wt_secret")); err != nil {
		t.Fatal(err)
	}

	ReportPanic(&sentinelErr{msg: "panicked globally"})
	Flush()

	reqs := c.requests()
	if len(reqs) != 1 || len(reqs[0].Events) != 1 {
		t.Fatalf("expected 1 event, got %d requests / %d events", len(reqs), len(reqs[0].Events))
	}
	if got := reqs[0].Events[0].Message; got != "panicked globally" {
		t.Errorf("message = %q", got)
	}
	if reqs[0].Events[0].StackTrace == "" {
		t.Error("expected a panic stack trace")
	}

	// Restore a clean global state (disabled client, no goroutines).
	if err := Init(Config{Disable: true}); err != nil {
		t.Fatal(err)
	}
}

func TestGlobalReportPanicWithoutInitIsNoop(t *testing.T) {
	// Ensure no global client is set.
	if err := Init(Config{Disable: true}); err != nil {
		t.Fatal(err)
	}
	ReportPanic(errors.New("ignored")) // must not panic
}

func TestInitReplacesAndClosesPrevious(t *testing.T) {
	c1 := newCapture(t, "wt_secret")
	c2 := newCapture(t, "wt_secret")

	if err := Init(testConfig(c1, "wt_secret")); err != nil {
		t.Fatal(err)
	}
	first := defaultClient()
	if first == nil {
		t.Fatal("Init did not set the global client")
	}

	if err := Init(testConfig(c2, "wt_secret")); err != nil {
		t.Fatal(err)
	}
	second := defaultClient()
	if second == first {
		t.Fatal("Init should replace the previous global client")
	}
	if !first.batch.closed {
		t.Error("previous client should be closed after re-Init")
	}

	// Restore a clean global state (disabled client, no goroutines).
	if err := Init(Config{Disable: true}); err != nil {
		t.Fatal(err)
	}
}

// flakySender fails the first n calls, then succeeds.
type flakySender struct {
	fails int64
	calls int64
}

func (f *flakySender) Send(_ context.Context, events []Event) ([]Result, error) {
	n := atomic.AddInt64(&f.calls, 1)
	if n <= f.fails {
		return nil, errors.New("transient failure")
	}
	res := make([]Result, len(events))
	for i := range res {
		res[i] = Result{IssueID: int64(i + 1)}
	}
	return res, nil
}

func TestFlushSucceedsWithoutRetry(t *testing.T) {
	f := &flakySender{}
	b := newBatch(f, time.Hour, 100, 1000, time.Second, 2, time.Millisecond, discardLogger())
	b.enqueue(Event{Message: "boom"})
	b.flush()
	if got := atomic.LoadInt64(&f.calls); got != 1 {
		t.Errorf("expected 1 attempt, got %d", got)
	}
	b.Close()
}

func TestFlushRetriesFailedSend(t *testing.T) {
	f := &flakySender{fails: 2}
	b := newBatch(f, time.Hour, 100, 1000, time.Second, 5, time.Millisecond, discardLogger())
	b.enqueue(Event{Message: "boom"})
	b.flush()
	if got := atomic.LoadInt64(&f.calls); got != 3 {
		t.Errorf("expected 3 attempts, got %d", got)
	}
	b.Close()
}

func TestFlushDropsAfterMaxRetries(t *testing.T) {
	f := &flakySender{fails: 99}
	b := newBatch(f, time.Hour, 100, 1000, time.Second, 2, time.Millisecond, discardLogger())
	b.enqueue(Event{Message: "boom"})
	b.flush()
	if got := atomic.LoadInt64(&f.calls); got != 3 {
		t.Errorf("expected 3 attempts (1 send + 2 retries), got %d", got)
	}
	b.Close()
}
