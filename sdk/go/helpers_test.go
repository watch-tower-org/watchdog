package wt

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// request is the server-side view of a reported event.
type request struct {
	Method string
	Path   string
	APIKey string
	Events []struct {
		Message    string         `json:"message"`
		ErrorType  string         `json:"error_type"`
		StackTrace string         `json:"stack_trace"`
		Project    string         `json:"project"`
		Tag        string         `json:"tag"`
		Context    map[string]any `json:"context"`
		Timestamp  *time.Time     `json:"timestamp"`
	}
}

// capture is a test server that records ingestion calls and returns a canned
// response per request.
type capture struct {
	srv *httptest.Server

	mu       sync.Mutex
	reqs     []request
	sent     int64
	failWith int // if >0, respond with this status instead of success
}

func newCapture(t *testing.T, key string) *capture {
	t.Helper()
	c := &capture{}
	c.srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Serve the version endpoint so NewClient's init-time verification
		// passes against this server.
		if r.URL.Path == versionBasePath {
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"success":true,"code":200,"message":"version retrieved","data":{"product":"watchdog","version":"test"}}`))
			return
		}

		var evts []struct {
			Message    string         `json:"message"`
			ErrorType  string         `json:"error_type"`
			StackTrace string         `json:"stack_trace"`
			Project    string         `json:"project"`
			Tag        string         `json:"tag"`
			Context    map[string]any `json:"context"`
			Timestamp  *time.Time     `json:"timestamp"`
		}
		_ = json.NewDecoder(r.Body).Decode(&evts)

		c.mu.Lock()
		fail := c.failWith
		if fail == 0 {
			c.reqs = append(c.reqs, request{
				Method: r.Method,
				Path:   r.URL.Path,
				APIKey: r.Header.Get("X-Api-Key"),
				Events: evts,
			})
			atomic.AddInt64(&c.sent, int64(len(evts)))
		}
		c.mu.Unlock()

		if fail > 0 {
			w.WriteHeader(fail)
			_, _ = w.Write([]byte(`{"error":"boom"}`))
			return
		}

		results := make([]map[string]any, len(evts))
		for i := range evts {
			results[i] = map[string]any{
				"issue_id":       int64(i + 1),
				"event_id":       int64(i + 1),
				"is_new_issue":   true,
				"was_regression": false,
				"fingerprint":    "abc",
			}
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"success":true,"code":201,"data":{"received":` + itoa(len(evts)) + `,"events":` + mustJSON(results) + `}}`))
	}))
	t.Cleanup(c.srv.Close)
	return c
}

func (c *capture) requests() []request {
	c.mu.Lock()
	defer c.mu.Unlock()
	out := make([]request, len(c.reqs))
	copy(out, c.reqs)
	return out
}

func (c *capture) count() int64 { return atomic.LoadInt64(&c.sent) }

// eventually polls fn until it returns true or the timeout elapses.
func eventually(t *testing.T, d time.Duration, fn func() bool) {
	t.Helper()
	deadline := time.Now().Add(d)
	for time.Now().Before(deadline) {
		if fn() {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatal("condition not met within timeout")
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	b := make([]byte, 0, 8)
	for n > 0 {
		b = append([]byte{byte('0' + n%10)}, b...)
		n /= 10
	}
	return string(b)
}

func mustJSON(v any) string {
	b, _ := json.Marshal(v)
	return string(b)
}
