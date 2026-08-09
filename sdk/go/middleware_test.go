package wt

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestReportPanicReportsPanicValue(t *testing.T) {
	c := newCapture(t, "wt_secret")
	cfg := testConfig(c, "wt_secret")
	cfg.BatchInterval = time.Hour
	cl, _ := NewClient(cfg)
	defer cl.Close()

	cl.ReportPanic("nil map access",
		WithTag("recovered"),
		WithContext(map[string]any{"handler": "checkout"}))
	cl.Flush()

	reqs := c.requests()
	if len(reqs) != 1 {
		t.Fatalf("expected 1 report, got %d", len(reqs))
	}
	e := reqs[0].Events[0]
	if e.Message != "nil map access" {
		t.Errorf("message = %q, want panic value", e.Message)
	}
	if e.Tag != "recovered" {
		t.Errorf("tag = %q", e.Tag)
	}
	if e.Context["handler"] != "checkout" {
		t.Errorf("context = %v", e.Context)
	}
	if e.StackTrace == "" {
		t.Error("expected a non-empty panic stack trace")
	}
	if strings.Contains(e.StackTrace, sdkPackagePrefix) {
		t.Errorf("panic stack should not contain SDK frames:\n%s", e.StackTrace)
	}
}

func TestRecoverMiddlewareReportsAndReturns500(t *testing.T) {
	c := newCapture(t, "wt_secret")
	cfg := testConfig(c, "wt_secret")
	cfg.BatchInterval = time.Hour
	cl, _ := NewClient(cfg)
	defer cl.Close()

	handler := cl.RecoverMiddleware()(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		panic("kaboom")
	}))

	req := httptest.NewRequest(http.MethodGet, "/orders/42", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Errorf("status = %d, want 500", rec.Code)
	}
	cl.Flush()

	reqs := c.requests()
	if len(reqs) != 1 {
		t.Fatalf("expected 1 report, got %d", len(reqs))
	}
	e := reqs[0].Events[0]
	if e.Message != "kaboom" {
		t.Errorf("message = %q, want panic value", e.Message)
	}
	if e.Context["url"] != "/orders/42" || e.Context["method"] != "GET" {
		t.Errorf("request context = %v", e.Context)
	}
	if e.StackTrace == "" {
		t.Error("expected a non-empty panic stack trace")
	}
	if strings.Contains(e.StackTrace, sdkPackagePrefix) {
		t.Errorf("panic stack should not contain SDK frames:\n%s", e.StackTrace)
	}
}

func TestRecoverMiddlewareKeepsServing(t *testing.T) {
	c := newCapture(t, "wt_secret")
	cfg := testConfig(c, "wt_secret")
	cfg.BatchInterval = time.Hour
	cl, _ := NewClient(cfg)
	defer cl.Close()

	var hits int
	handler := cl.RecoverMiddleware()(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		hits++
		if hits == 1 {
			panic("boom")
		}
		w.WriteHeader(http.StatusOK)
	}))

	for i := 0; i < 2; i++ {
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))
	}
	if hits != 2 {
		t.Errorf("server stopped after panic, hits = %d", hits)
	}
}

func TestRecoverHandlerReportsAndRepanics(t *testing.T) {
	c := newCapture(t, "wt_secret")
	cfg := testConfig(c, "wt_secret")
	cfg.BatchInterval = time.Hour
	cl, _ := NewClient(cfg)
	defer cl.Close()

	wrapped := cl.RecoverHandler(func() {
		panic(errors.New("async failure"))
	})

	var repanicked any
	func() {
		defer func() { repanicked = recover() }()
		wrapped()
	}()
	if repanicked == nil {
		t.Fatal("RecoverHandler should re-panic")
	}
	cl.Flush()

	if c.count() != 1 {
		t.Errorf("expected 1 report, got %d", c.count())
	}
}

func TestRecoverHandlerNilStillCallsFn(t *testing.T) {
	c := newCapture(t, "wt_secret")
	cfg := testConfig(c, "wt_secret")
	cfg.BatchInterval = time.Hour
	cl, _ := NewClient(cfg)
	defer cl.Close()

	called := false
	cl.RecoverHandler(func() { called = true })()
	if !called {
		t.Error("RecoverHandler should call fn")
	}
}

func TestCapturePanicStackTrimsSDKFrames(t *testing.T) {
	raw := `goroutine 7 [running]:
runtime/debug.Stack()
	/usr/local/go/src/runtime/debug/stack.go:26 +0x64
github.com/watch-tower-org/watchtower/sdk/go.capturePanicStack()
	/home/wt/sdk/go/capture.go:58 +0x3c
github.com/watch-tower-org/watchtower/sdk/go.(*Client).reportPanic(0x1)
	/home/wt/sdk/go/watchtower.go:0 +0x0
main.handler.func1()
	/app/main.go:42 +0x1a
net/http.HandlerFunc.ServeHTTP()
	/usr/local/go/src/net/http/server.go:2141 +0x29
`
	got := trimPanicStack(raw)
	if strings.Contains(got, "watchtower/sdk/go") {
		t.Errorf("SDK frames not trimmed:\n%s", got)
	}
	if strings.Contains(got, "debug.Stack") {
		t.Errorf("debug.Stack frame not trimmed:\n%s", got)
	}
	if !strings.Contains(got, "main.handler.func1") {
		t.Errorf("user frame missing after trim:\n%s", got)
	}
	if strings.Contains(got, "goroutine 7") {
		t.Errorf("goroutine header not trimmed:\n%s", got)
	}
}
