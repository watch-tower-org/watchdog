package wt

import (
	"errors"
	"testing"
	"time"
)

func TestFlushOnInterval(t *testing.T) {
	c := newCapture(t, "wt_secret")
	cfg := testConfig(c, "wt_secret")
	cfg.BatchInterval = 30 * time.Millisecond
	cl, _ := NewClient(cfg)
	defer cl.Close()

	cl.Report(errors.New("e1"))
	eventually(t, time.Second, func() bool { return c.count() >= 1 })
}

func TestFlushOnSize(t *testing.T) {
	c := newCapture(t, "wt_secret")
	cfg := testConfig(c, "wt_secret")
	cfg.BatchSize = 3
	cfg.BatchInterval = time.Hour
	cl, _ := NewClient(cfg)
	defer cl.Close()

	for i := 0; i < 3; i++ {
		cl.Report(errors.New("e"))
	}
	eventually(t, time.Second, func() bool { return c.count() >= 3 })
}

func TestBatchingGroupsEvents(t *testing.T) {
	c := newCapture(t, "wt_secret")
	cfg := testConfig(c, "wt_secret")
	cfg.BatchSize = 5
	cfg.BatchInterval = time.Hour
	cl, _ := NewClient(cfg)
	defer cl.Close()

	for i := 0; i < 5; i++ {
		cl.Report(errors.New("e"))
	}
	eventually(t, time.Second, func() bool { return len(c.requests()) == 1 })

	reqs := c.requests()
	if len(reqs) != 1 {
		t.Fatalf("expected a single batched request, got %d", len(reqs))
	}
	if len(reqs[0].Events) != 5 {
		t.Errorf("expected 5 events in one request, got %d", len(reqs[0].Events))
	}
}

func TestDropOnFullQueue(t *testing.T) {
	c := newCapture(t, "wt_secret")
	cfg := testConfig(c, "wt_secret")
	cfg.MaxQueueSize = 2
	cfg.BatchSize = 100
	cfg.BatchInterval = time.Hour
	cl, _ := NewClient(cfg)
	defer cl.Close()

	for i := 0; i < 5; i++ {
		cl.Report(errors.New("e"))
	}
	cl.Flush()
	if got := c.count(); got != 2 {
		t.Errorf("expected only the 2 buffered events to be sent, got %d", got)
	}
}

func TestFailedSendDropsBatch(t *testing.T) {
	c := newCapture(t, "wt_secret")
	c.failWith = 503
	cfg := testConfig(c, "wt_secret")
	cl, _ := NewClient(cfg)

	cl.Report(errors.New("e1"))
	cl.Close()
	if c.count() != 0 {
		t.Errorf("failed batch should be dropped, sent %d", c.count())
	}
}

func TestCloseFlushesAll(t *testing.T) {
	c := newCapture(t, "wt_secret")
	cfg := testConfig(c, "wt_secret")
	cfg.BatchInterval = time.Hour
	cl, _ := NewClient(cfg)

	for i := 0; i < 4; i++ {
		cl.Report(errors.New("e"))
	}
	cl.Close()
	if got := c.count(); got != 4 {
		t.Errorf("expected 4 events on Close, got %d", got)
	}
}
