package uptime

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/watch-tower-org/watchtower/backend/internal/model"
)

func TestRunCheck_Up(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	}))
	defer srv.Close()

	result := runCheck(t.Context(), srv.URL, 5*time.Second)

	if result.Status != model.ServiceStatusUp {
		t.Fatalf("expected up, got %v", result.Status)
	}
	if result.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", result.StatusCode)
	}
	if result.Error != nil {
		t.Fatalf("expected no error, got %v", *result.Error)
	}
	if result.ResponseTimeMs < 0 {
		t.Fatalf("expected non-negative latency, got %d", result.ResponseTimeMs)
	}
}

func TestRunCheck_Down_5xx(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	result := runCheck(t.Context(), srv.URL, 5*time.Second)

	if result.Status != model.ServiceStatusDown {
		t.Fatalf("expected down, got %v", result.Status)
	}
	if result.StatusCode != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", result.StatusCode)
	}
	if result.Error != nil {
		t.Fatalf("expected no error for 500, got %v", *result.Error)
	}
}

func TestRunCheck_Down_ConnectionError(t *testing.T) {
	// Server never starts; connect is refused.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	url := srv.URL
	srv.Close()

	result := runCheck(t.Context(), url, time.Second)

	if result.Status != model.ServiceStatusDown {
		t.Fatalf("expected down, got %v", result.Status)
	}
	if result.Error == nil {
		t.Fatal("expected an error for connection failure")
	}
}

func TestRunCheck_Timeout(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(2 * time.Second)
	}))
	defer srv.Close()

	result := runCheck(t.Context(), srv.URL, 100*time.Millisecond)

	if result.Status != model.ServiceStatusDown {
		t.Fatalf("expected down on timeout, got %v", result.Status)
	}
	if result.Error == nil {
		t.Fatal("expected an error for timeout")
	}
}

func TestRunCheck_RecordsLatency(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(50 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	result := runCheck(t.Context(), srv.URL, 5*time.Second)

	if result.Status != model.ServiceStatusUp {
		t.Fatalf("expected up, got %v", result.Status)
	}
	if result.ResponseTimeMs < 50 {
		t.Fatalf("expected latency >= 50ms, got %dms", result.ResponseTimeMs)
	}
}
