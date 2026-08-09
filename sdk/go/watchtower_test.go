package wt

import (
	"errors"
	"net/http"
	"net/http/httptest"
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

func TestNewClientRejectsMalformedBaseURL(t *testing.T) {
	c := newCapture(t, "k")
	valid := c.srv.URL

	cases := []struct {
		name    string
		baseURL string
		wantErr bool
	}{
		{name: "valid http", baseURL: valid, wantErr: false},
		{name: "whitespace padded", baseURL: "  " + valid + "  ", wantErr: false},
		{name: "missing scheme", baseURL: "localhost:8080", wantErr: true},
		{name: "wrong scheme", baseURL: "ftp://watchtower.example.com", wantErr: true},
		{name: "no host", baseURL: "http://", wantErr: true},
		{name: "unparsable", baseURL: "://bad", wantErr: true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			cfg := DefaultConfig()
			cfg.BaseURL = tc.baseURL
			cfg.APIKey = "k"
			cfg.Project = "p"
			cl, err := NewClient(cfg)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("expected error for BaseURL %q", tc.baseURL)
				}
				if cl != nil {
					cl.Close()
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error for BaseURL %q: %v", tc.baseURL, err)
			}
			cl.Close()
		})
	}
}

func TestNewClientVerifiesBaseURL(t *testing.T) {
	t.Run("unreachable", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
		url := srv.URL
		srv.Close()

		cfg := DefaultConfig()
		cfg.BaseURL = url
		cfg.APIKey = "k"
		cfg.Project = "p"
		if _, err := NewClient(cfg); err == nil {
			t.Fatal("expected error for unreachable BaseURL")
		}
	})

	t.Run("wrong product", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"success":true,"code":200,"message":"ok","data":{"product":"not-watchdog","version":"1.0.0"}}`))
		}))
		defer srv.Close()

		cfg := DefaultConfig()
		cfg.BaseURL = srv.URL
		cfg.APIKey = "k"
		cfg.Project = "p"
		_, err := NewClient(cfg)
		if err == nil {
			t.Fatal("expected error for non-WatchTower product")
		}
		if !strings.Contains(err.Error(), "not a WatchTower instance") {
			t.Errorf("error = %v, want 'not a WatchTower instance'", err)
		}
	})

	t.Run("unauthorized key", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusUnauthorized)
		}))
		defer srv.Close()

		cfg := DefaultConfig()
		cfg.BaseURL = srv.URL
		cfg.APIKey = "bad-key"
		cfg.Project = "p"
		_, err := NewClient(cfg)
		if err == nil {
			t.Fatal("expected error for unauthorized key")
		}
		if !strings.Contains(err.Error(), "invalid or revoked") {
			t.Errorf("error = %v, want 'invalid or revoked API key'", err)
		}
	})

	t.Run("verify disabled", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
		url := srv.URL
		srv.Close()

		cfg := DefaultConfig()
		cfg.BaseURL = url
		cfg.APIKey = "k"
		cfg.Project = "p"
		cfg.VerifyBaseURL = false
		cl, err := NewClient(cfg)
		if err != nil {
			t.Fatalf("VerifyBaseURL=false should skip the network check, got %v", err)
		}
		cl.Close()
	})
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
