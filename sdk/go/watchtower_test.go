package wt

import (
	"errors"
	"strings"
	"testing"
	"time"
)

func testConfig(c *capture, key string) Config {
	cfg := DefaultConfig()
	cfg.BaseURL = c.srv.URL
	cfg.APIKey = key
	cfg.Project = "payments-api"
	cfg.Tag = "production"
	cfg.BatchInterval = time.Hour // prevent auto-flush; tests flush explicitly
	return cfg
}

func TestReportSendsBatchedPayload(t *testing.T) {
	c := newCapture(t, "wt_secret")
	cl, err := NewClient(testConfig(c, "wt_secret"))
	if err != nil {
		t.Fatal(err)
	}
	defer cl.Close()

	ts := time.Date(2026, 8, 8, 12, 0, 0, 0, time.UTC)
	cl.Report(errors.New("database timeout: conn refused"),
		WithTag("worker"),
		WithContext(map[string]any{"job": "cron", "attempt": 2}),
		WithTimestamp(ts),
	)
	cl.Flush()

	reqs := c.requests()
	if len(reqs) != 1 {
		t.Fatalf("expected 1 request, got %d", len(reqs))
	}
	r := reqs[0]
	if r.Method != "POST" {
		t.Errorf("method = %s, want POST", r.Method)
	}
	if r.Path != "/api/watchtower/v1/events" {
		t.Errorf("path = %s", r.Path)
	}
	if r.APIKey != "wt_secret" {
		t.Errorf("api key = %q", r.APIKey)
	}
	if len(r.Events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(r.Events))
	}
	e := r.Events[0]
	if e.Message != "database timeout: conn refused" {
		t.Errorf("message = %q", e.Message)
	}
	if e.Project != "payments-api" {
		t.Errorf("project = %q", e.Project)
	}
	if e.Tag != "worker" {
		t.Errorf("tag = %q, want per-report override", e.Tag)
	}
	if e.Context["job"] != "cron" || e.Context["attempt"] != float64(2) {
		t.Errorf("context = %v", e.Context)
	}
	if e.Context["hostname"] == "" {
		t.Error("expected hostname in context")
	}
	if e.Timestamp == nil || !e.Timestamp.Equal(ts) {
		t.Errorf("timestamp = %v, want %v", e.Timestamp, ts)
	}
	if e.StackTrace == "" {
		t.Error("expected a non-empty stack trace")
	}
	if strings.Contains(e.StackTrace, sdkPackagePrefix) {
		t.Errorf("stack trace should not contain SDK frames:\n%s", e.StackTrace)
	}
}

func TestReportNilIsNoop(t *testing.T) {
	c := newCapture(t, "wt_secret")
	cl, _ := NewClient(testConfig(c, "wt_secret"))
	defer cl.Close()
	cl.Report(nil)
	cl.Flush()
	if c.count() != 0 {
		t.Errorf("expected no events, sent %d", c.count())
	}
}

func TestReportSyncReturnsResults(t *testing.T) {
	c := newCapture(t, "wt_secret")
	cl, _ := NewClient(testConfig(c, "wt_secret"))
	defer cl.Close()

	res, err := cl.ReportSync(errors.New("boom"))
	if err != nil {
		t.Fatal(err)
	}
	if len(res) != 1 || res[0].IssueID != 1 || !res[0].IsNewIssue {
		t.Errorf("results = %+v", res)
	}
}

func TestReportSyncServerError(t *testing.T) {
	c := newCapture(t, "wt_secret")
	c.failWith = 500
	cl, _ := NewClient(testConfig(c, "wt_secret"))
	defer cl.Close()

	_, err := cl.ReportSync(errors.New("boom"))
	if err == nil {
		t.Fatal("expected error from 500 response")
	}
	if !strings.Contains(err.Error(), "500") {
		t.Errorf("error should mention status, got %v", err)
	}
}

func TestSampleRateZeroReportsNothing(t *testing.T) {
	c := newCapture(t, "wt_secret")
	cfg := testConfig(c, "wt_secret")
	cfg.SampleRate = 0
	cl, _ := NewClient(cfg)
	defer cl.Close()

	cl.Report(errors.New("should be dropped"))
	cl.Flush()
	if c.count() != 0 {
		t.Errorf("sample rate 0 should send nothing, sent %d", c.count())
	}
}

func TestNewClientValidation(t *testing.T) {
	cfg := DefaultConfig()
	cfg.BaseURL = "http://x"
	cfg.APIKey = "k"
	if _, err := NewClient(cfg); err == nil {
		t.Error("expected error when Project missing")
	}
}

func TestDisabledClientNoop(t *testing.T) {
	c := newCapture(t, "wt_secret")
	cfg := testConfig(c, "wt_secret")
	cfg.Disable = true
	cl, _ := NewClient(cfg)
	defer cl.Close()

	cl.Report(errors.New("ignored"))
	cl.Flush()
	if c.count() != 0 {
		t.Errorf("disabled client should not send, sent %d", c.count())
	}
}

func TestReportAfterCloseDropped(t *testing.T) {
	c := newCapture(t, "wt_secret")
	cl, _ := NewClient(testConfig(c, "wt_secret"))

	cl.Report(errors.New("sent"))
	cl.Close()
	if c.count() != 1 {
		t.Errorf("Close should flush pending events, sent %d", c.count())
	}
	cl.Report(errors.New("dropped"))
	cl.Close()
	if c.count() != 1 {
		t.Errorf("reports after Close should be dropped, sent %d", c.count())
	}
}
