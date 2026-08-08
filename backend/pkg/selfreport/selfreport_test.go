package selfreport

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	wt "github.com/watch-tower-org/watchdog/sdk/go"

	"github.com/watch-tower-org/watchdog/backend/internal/config"
	"github.com/watch-tower-org/watchdog/backend/internal/model"
)

// stubIngester records what sink.Send hands over.
type stubIngester struct {
	mu      sync.Mutex
	reqs    []*model.IngestEventRequest
	fail    error
	results []*model.IngestResult
}

func (s *stubIngester) IngestBatch(_ context.Context, reqs []*model.IngestEventRequest) ([]*model.IngestResult, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.reqs = append(s.reqs, reqs...)
	if s.fail != nil {
		return nil, s.fail
	}
	return s.results, nil
}

func (s *stubIngester) seen() []*model.IngestEventRequest {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]*model.IngestEventRequest(nil), s.reqs...)
}

func TestSinkConvertsEventsToIngestRequests(t *testing.T) {
	stub := &stubIngester{results: []*model.IngestResult{{IssueID: 7, EventID: 9, IsNewIssue: true}}}
	s := &sink{ingest: stub}

	ts := "2026-08-08T00:00:00Z"
	results, err := s.Send(context.Background(), []wt.Event{{
		Message:    "boom",
		ErrorType:  "runtime.errorString",
		StackTrace: "main.foo\n\t/main.go:10",
		Project:    "watchtower-self",
		Tag:        "log.error",
		Context:    map[string]any{"url": "/x"},
		Timestamp:  mustTimePtr(t, ts),
	}})
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 1 || results[0].IssueID != 7 || !results[0].IsNewIssue {
		t.Fatalf("results = %+v", results)
	}
	reqs := stub.seen()
	if len(reqs) != 1 {
		t.Fatalf("ingest reqs = %d", len(reqs))
	}
	r := reqs[0]
	if r.Message != "boom" || r.ErrorType != "runtime.errorString" || r.StackTrace != "main.foo\n\t/main.go:10" {
		t.Errorf("req = %+v", r)
	}
	if r.Project != "watchtower-self" || r.Tag != "log.error" || r.Context["url"] != "/x" {
		t.Errorf("req = %+v", r)
	}
	if r.Timestamp == nil || r.Timestamp.Format("2006-01-02T15:04:05Z07:00") != ts {
		t.Errorf("timestamp = %v", r.Timestamp)
	}
}

func TestSinkEmptyBatchIsNoOp(t *testing.T) {
	stub := &stubIngester{}
	s := &sink{ingest: stub}
	results, err := s.Send(context.Background(), nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 0 || len(stub.seen()) != 0 {
		t.Fatalf("results=%v reqs=%v", results, stub.seen())
	}
}

func TestSinkReturnsErrorAndNoResultsOnFailure(t *testing.T) {
	stub := &stubIngester{fail: errors.New("db down")}
	s := &sink{ingest: stub}
	_, err := s.Send(context.Background(), []wt.Event{{Message: "x"}})
	if err == nil {
		t.Fatal("expected error from ingest to propagate")
	}
}

func TestSinkReentrancyGuardClearsAfterIngest(t *testing.T) {
	stub := &stubIngester{results: []*model.IngestResult{{IssueID: 1}}}
	s := &sink{ingest: stub}
	if s.inSelf.Load() {
		t.Fatal("guard should start false")
	}
	if _, err := s.Send(context.Background(), []wt.Event{{Message: "x"}}); err != nil {
		t.Fatal(err)
	}
	if s.inSelf.Load() {
		t.Fatal("guard should be cleared after Send returns")
	}
}

func TestReporterNilSafe(t *testing.T) {
	var r *Reporter
	r.Report(errors.New("boom"))
	r.ReportPanic("boom")
	r.Recover("boom", map[string]any{"url": "/x"})
	r.Close()
}

func TestReporterDisabledReturnsNil(t *testing.T) {
	cfg := config.SelfReportConfig{Enabled: false}
	r, err := New(cfg, nil)
	if err != nil {
		t.Fatal(err)
	}
	if r != nil {
		t.Fatalf("expected nil reporter, got %+v", r)
	}
}

func mustTimePtr(t *testing.T, s string) *time.Time {
	t.Helper()
	parsed, err := time.Parse(time.RFC3339, s)
	if err != nil {
		t.Fatal(err)
	}
	return &parsed
}
