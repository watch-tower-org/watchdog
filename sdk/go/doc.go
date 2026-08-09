// Package wt provides a lightweight, zero-dependency client for WatchTower, a
// self-hosted error-tracking service.
//
// # Getting started
//
// Call Init once at startup to configure a process-wide client, then Report
// errors as they happen. On shutdown, Close flushes any buffered events.
//
//	import wt "github.com/watch-tower-org/watchdog/sdk/go"
//
//	func main() {
//		if err := wt.Init(wt.Config{
//			BaseURL: "http://localhost:8080",
//			APIKey:  "wt_...", // create a key in the WatchTower dashboard
//			Project: "payments-api",
//			Tag:     "production",
//		}); err != nil {
//			log.Fatal(err)
//		}
//		defer wt.Close()
//	}
//
// Reporting is asynchronous and non-blocking:
//
//	wt.Report(err, wt.WithTag("checkout"), wt.WithContext(map[string]any{"order_id": "ord_1234"}))
//
// Use ReportSync for low-volume or test paths where you need the ingest result
// (issue ID, whether it is a new issue or a regression). Use ReportPanic from
// custom recovery code to report a recovered panic value with its panic-time
// stack. Use RecoverMiddleware in an net/http server to report panics and
// respond 500 without killing the process, and RecoverHandler to re-raise
// panics after reporting (Sentry-style semantics for worker/goroutine
// wrappers).
//
// # Delivery
//
// Events are buffered and flushed in batches (see DefaultConfig for the
// defaults: 5s interval, batch size 100, queue 5000, 10s HTTP timeout). Failed
// batches are dropped and logged to the configured Logger; the SDK never
// retries and never blocks your application.
//
// # Base-URL verification
//
// By default NewClient runs a live check against BaseURL's /version endpoint
// before starting. This fails fast with an error if the URL is unreachable,
// does not point to a WatchTower instance, or the API key is invalid or
// revoked. Set Config.VerifyBaseURL to false to skip the check (e.g. when the
// instance is only reachable later). The check is never performed when a custom
// Config.Sender is used.
//
// By default events are posted over HTTP to BaseURL using APIKey. Set
// Config.Sender to a custom Sender to override delivery entirely — e.g. to
// hand events to an in-process pipeline — in which case BaseURL and APIKey are
// not required. WatchTower's own self-reporting uses this to feed captured
// events straight into its local ingestion controller.
package wt
