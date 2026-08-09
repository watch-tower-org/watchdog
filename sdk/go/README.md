# WatchTower Go SDK

A lightweight, **zero-dependency** client for [WatchTower](https://github.com/watch-tower-org/watchtower), a self-hosted error-tracking service.

## Install

```bash
go get github.com/watch-tower-org/watchtower/sdk/go@latest
```

## Usage

```go
import wt "github.com/watch-tower-org/watchtower/sdk/go"

func main() {
    if err := wt.Init(wt.Config{
        BaseURL: "http://localhost:8080", // WatchTower instance
        APIKey:  "wt_...",                // create a key in the dashboard
        Project: "payments-api",
        Tag:     "production",
    }); err != nil {
        log.Fatal(err)
    }
    defer wt.Close() // flushes buffered events on shutdown

    // ...
}

// Async, never blocks, never panics.
wt.Report(err, wt.WithTag("checkout"), wt.WithContext(map[string]any{"order_id": "ord_1234"}))

// Sync, returns the ingest result (issue id, is-new-issue, was-regression).
results, err := wt.ReportSync(err)

// net/http recovery: report the panic and return 500.
http.ListenAndServe(":8080", wt.RecoverMiddleware()(mux))

// Worker/goroutine wrapper: report *and* re-raise (Sentry-style).
wt.RecoverHandler(func() { doWork() })
```

## Configuration

- `wt.DefaultConfig()` — sane defaults (5s flush interval, batch size 100, queue 5000, 10s HTTP timeout, 2 send retries).
- `wt.ConfigFromEnv()` — build a config from `WATCHTOWER_*` env vars (`WATCHTOWER_BASE_URL`, `WATCHTOWER_API_KEY`, `WATCHTOWER_PROJECT`, `WATCHTOWER_TAG`, `WATCHTOWER_RELEASE`, `WATCHTOWER_SAMPLE_RATE`).
- `BaseURL` must be an absolute `http`/`https` URL including a host; `NewClient` rejects malformed URLs with a descriptive error. Reachability is not checked at init, so a valid-but-unreachable URL only surfaces when a batch flush fails.
- `wt.Config.Sender` — override delivery with a custom `Sender` (`Send(ctx, []Event)`); `BaseURL`/`APIKey` then become optional. Used by WatchTower's own in-process self-reporting.
- `wt.Config.MaxRetries` — additional attempts for a failed background batch flush (default 2). Retries run in the flusher goroutine, so callers never block; `ReportSync` is always one-shot.

## Production notes

- **Always start from `DefaultConfig()` or `ConfigFromEnv()`.** A hand-built `wt.Config{...}` leaves `SampleRate` at `0`, which silently drops *every* event (the SDK warns at `NewClient` time). If you do want to disable reporting, prefer `Config.Disable`.
- Events are batched and flushed on an interval; a failed batch is retried up to `MaxRetries` times, then dropped and logged — the SDK never blocks your application. A crash between flushes can lose up to one batch interval of events; call `defer wt.Close()` on graceful shutdown to flush.
- The backend reports the Go type of each error as `error_type` by default; override per report with `WithErrorType`.
- Calling `wt.Init` again replaces and flushes the previous client (it does not silently ignore the new config).

A runnable example lives in [`example/`](example/main.go).
