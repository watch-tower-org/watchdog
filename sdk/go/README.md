# WatchTower Go SDK

A lightweight, **zero-dependency** client for [WatchTower](https://github.com/watch-tower-org/watchdog), a self-hosted error-tracking service.

## Install

```bash
go get github.com/watch-tower-org/watchdog/sdk/go@latest
```

## Usage

```go
import wt "github.com/watch-tower-org/watchdog/sdk/go"

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

- `wt.DefaultConfig()` — sane defaults (5s flush interval, batch size 100, queue 5000, 10s HTTP timeout).
- `wt.ConfigFromEnv()` — build a config from `WATCHTOWER_*` env vars (`WATCHTOWER_BASE_URL`, `WATCHTOWER_API_KEY`, `WATCHTOWER_PROJECT`, `WATCHTOWER_TAG`, `WATCHTOWER_RELEASE`, `WATCHTOWER_SAMPLE_RATE`).
- `wt.Config.Sender` — override delivery with a custom `Sender` (`Send(ctx, []Event)`); `BaseURL`/`APIKey` then become optional. Used by WatchTower's own in-process self-reporting.

## Behavior notes

- Events are batched and flushed on an interval; failed batches are dropped and logged (no retry, no blocking).
- A runnable example lives in [`example/`](example/main.go).
