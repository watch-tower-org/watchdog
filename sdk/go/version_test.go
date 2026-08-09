package wt

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestCheckVersion(t *testing.T) {
	cases := []struct {
		name        string
		handler     http.HandlerFunc
		key         string
		wantErr     bool
		errContains string
	}{
		{
			name: "valid instance",
			handler: func(w http.ResponseWriter, r *http.Request) {
				if got := r.Header.Get("X-Api-Key"); got != "k" {
					t.Errorf("X-Api-Key = %q, want k", got)
				}
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{"success":true,"code":200,"message":"version retrieved","data":{"product":"watchtower","version":"v0.1.4"}}`))
			},
			key: "k",
		},
		{
			name: "success false",
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{"success":false,"code":500,"message":"boom"}`))
			},
			key:         "k",
			wantErr:     true,
			errContains: "not a WatchTower instance",
		},
		{
			name: "unauthorized key",
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusUnauthorized)
			},
			key:         "bad",
			wantErr:     true,
			errContains: "invalid or revoked",
		},
		{
			name:        "unreachable",
			handler:     nil, // replaced with a closed server below
			key:         "k",
			wantErr:     true,
			errContains: "version check failed",
		},
		{
			name: "not json",
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "text/plain")
				_, _ = w.Write([]byte("<html>nope</html>"))
			},
			key:         "k",
			wantErr:     true,
			errContains: "not a WatchTower instance",
		},
		{
			name: "wrong product",
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{"success":true,"code":200,"message":"ok","data":{"product":"sentry","version":"1.0.0"}}`))
			},
			key:         "k",
			wantErr:     true,
			errContains: "not a WatchTower instance",
		},
		{
			name: "empty version accepted",
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{"success":true,"code":200,"message":"ok","data":{"product":"watchtower","version":""}}`))
			},
			key: "k",
		},
		{
			name: "server error",
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusInternalServerError)
			},
			key:         "k",
			wantErr:     true,
			errContains: "server returned 500",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var baseURL string
			if tc.handler == nil {
				srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
				baseURL = srv.URL
				srv.Close()
			} else {
				srv := httptest.NewServer(tc.handler)
				defer srv.Close()
				baseURL = srv.URL
			}

			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			_, err := CheckVersion(ctx, baseURL, tc.key)
			if tc.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				if tc.errContains != "" && !strings.Contains(err.Error(), tc.errContains) {
					t.Errorf("error = %q, want substring %q", err, tc.errContains)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}

func TestCheckVersionReturnsInfo(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"success":true,"code":200,"message":"version retrieved","data":{"product":"watchtower","version":"v0.1.4"}}`))
	}))
	defer srv.Close()

	info, err := CheckVersion(context.Background(), srv.URL, "k")
	if err != nil {
		t.Fatalf("CheckVersion: %v", err)
	}
	if info.Product != "watchtower" {
		t.Errorf("product = %q, want watchtower", info.Product)
	}
	if info.Version != "v0.1.4" {
		t.Errorf("version = %q, want v0.1.4", info.Version)
	}
}
