package wt

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"
)

// fakeSender records every delivered batch.
type fakeSender struct {
	mu     sync.Mutex
	batches [][]Event
	err    error
}

func (f *fakeSender) Send(_ context.Context, events []Event) ([]Result, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	cp := make([]Event, len(events))
	copy(cp, events)
	f.batches = append(f.batches, cp)
	if f.err != nil {
		return nil, f.err
	}
	results := make([]Result, len(events))
	for i := range events {
		results[i] = Result{IssueID: int64(i + 1), EventID: int64(i + 1), IsNewIssue: true}
	}
	return results, nil
}

func (f *fakeSender) received() []Event {
	f.mu.Lock()
	defer f.mu.Unlock()
	var out []Event
	for _, b := range f.batches {
		out = append(out, b...)
	}
	return out
}

func TestClientWithSenderSendsBatchedEvents(t *testing.T) {
	sink := &fakeSender{}
	cfg := DefaultConfig()
	cfg.Project = "payments-api"
	cfg.BatchInterval = time.Hour
	cfg.Sender = sink

	cl, err := NewClient(cfg)
	if err != nil {
		t.Fatal(err)
	}
	defer cl.Close()

	cl.Report(errors.New("db timeout"),
		WithTag("worker"),
		WithContext(map[string]any{"job": "cron"}))
	cl.Flush()

	got := sink.received()
	if len(got) != 1 {
		t.Fatalf("expected 1 event delivered, got %d", len(got))
	}
	e := got[0]
	if e.Message != "db timeout" || e.Project != "payments-api" || e.Tag != "worker" {
		t.Errorf("event = %+v", e)
	}
	if e.Context["job"] != "cron" {
		t.Errorf("context = %v", e.Context)
	}
	if e.StackTrace == "" {
		t.Error("expected captured stack trace")
	}
}

func TestClientWithSenderReportSync(t *testing.T) {
	sink := &fakeSender{}
	cfg := DefaultConfig()
	cfg.Project = "payments-api"
	cfg.Sender = sink

	cl, err := NewClient(cfg)
	if err != nil {
		t.Fatal(err)
	}
	defer cl.Close()

	results, err := cl.ReportSync(errors.New("sync boom"))
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 1 || results[0].IssueID != 1 {
		t.Fatalf("results = %+v", results)
	}
	if got := sink.received(); len(got) != 1 || got[0].Message != "sync boom" {
		t.Fatalf("delivered = %+v", got)
	}
}

func TestNewClientSenderSkipsHTTPValidation(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Project = "payments-api"
	cfg.Sender = &fakeSender{} // no BaseURL, no APIKey

	if _, err := NewClient(cfg); err != nil {
		t.Fatalf("client with Sender should not require BaseURL/APIKey: %v", err)
	}
}

func TestNewClientStillRequiresProjectWithSender(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Sender = &fakeSender{}

	if _, err := NewClient(cfg); err == nil {
		t.Fatal("expected error when Project is empty")
	}
}

func TestNewClientRequiresHTTPConfigWithoutSender(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Project = "payments-api"

	if _, err := NewClient(cfg); err == nil {
		t.Fatal("expected error when BaseURL/APIKey missing and no Sender")
	}
}
