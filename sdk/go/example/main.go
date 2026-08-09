// Command example demonstrates the WatchTower Go SDK. Run it with the
// backend serving on localhost:8080 and a valid API key:
//
//	go run . -api-key wt_xxx [-project payments-api]
package main

import (
	"flag"
	"log"
	"net/http"
	"os"
	"time"

	wt "github.com/watch-tower-org/watchtower/sdk/go"
)

func main() {
	var (
		apiKey  = flag.String("api-key", os.Getenv("WATCHTOWER_API_KEY"), "WatchTower API key")
		baseURL = flag.String("base-url", "http://localhost:8080", "WatchTower base URL")
		project = flag.String("project", os.Getenv("WATCHTOWER_PROJECT"), "project label")
	)
	flag.Parse()

	if *apiKey == "" {
		log.Fatal("api key is required (pass -api-key or set WATCHTOWER_API_KEY)")
	}

	cfg := wt.DefaultConfig()
	cfg.BaseURL = *baseURL
	cfg.APIKey = *apiKey
	cfg.Project = *project
	cfg.Tag = "example"
	cfg.BatchInterval = 2 * time.Second

	if err := wt.Init(cfg); err != nil {
		log.Fatalf("init: %v", err)
	}
	defer wt.Close()

	// 1. Manual error report with tag + context.
	log.Println("reporting a manual error…")
	wt.Report(&dbError{op: "connect", msg: "connection refused"},
		wt.WithTag("checkout"),
		wt.WithContext(map[string]any{"order_id": "ord_1234", "attempt": 3}))

	// 2. Synchronous report returns the backend result.
	res, err := wt.ReportSync(&dbError{op: "query", msg: "deadlock detected"})
	if err != nil {
		log.Printf("report sync error: %v", err)
	} else if len(res) > 0 {
		log.Printf("issue #%d event #%d (new=%v)", res[0].IssueID, res[0].EventID, res[0].IsNewIssue)
	}

	// 3. HTTP server with panic recovery middleware.
	handler := wt.RecoverMiddleware()(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/boom":
			panic("unexpected state: nil supplier")
		case "/ok":
			w.WriteHeader(http.StatusOK)
		default:
			http.NotFound(w, r)
		}
	}))

	addr := "127.0.0.1:9099"
	go func() {
		log.Printf("panic demo server on http://%s (hit /boom)", addr)
		if err := http.ListenAndServe(addr, handler); err != nil {
			log.Printf("demo server stopped: %v", err)
		}
	}()

	// Trigger the panic so the report is captured.
	time.Sleep(500 * time.Millisecond)
	resp, err := http.Get("http://" + addr + "/boom")
	if err != nil {
		log.Printf("boom request failed: %v", err)
	} else {
		resp.Body.Close()
		log.Printf("GET /boom -> %s", resp.Status)
	}

	// Give the flusher a moment, then drain.
	time.Sleep(3 * time.Second)
	wt.Flush()
	log.Println("done")
}

// dbError is a typical wrapped error; its message carries the error type the
// backend derives for fingerprinting ("dbError: ...").
type dbError struct {
	op  string
	msg string
}

func (e *dbError) Error() string { return "dbError: " + e.op + ": " + e.msg }
