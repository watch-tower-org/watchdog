# WatchTower

Self-hosted error tracking and alerting for Go microservices — a lightweight,
self-hosted Sentry. The Go SDK captures errors and panics, batches them over
HTTP, the backend deduplicates them into issues, and a dashboard + throttled
email alerts keep you on top of what's breaking.

One static binary embeds the React dashboard. Postgres backs everything.

## Features

- **Go SDK** — stdlib-only (zero dependencies), client-side batching, panic
  recovery middleware, sync + async reporting.
- **Deduplication** — repeated errors are fingerprinted and grouped into
  issues with occurrence counts and regression tracking.
- **Dashboard** — embedded React SPA: issues, events, API keys, alert rules,
  alert history, recipient lists, and settings.
- **Alerting** — configurable, throttled email alerts on new issues,
  regressions, and spikes.
- **API keys** — with optional per-project scoping.
- **Self-reporting (dogfooding)** — the backend reports its own errors
  through the exact same pipeline it exposes to your services.
- **Simple hosting** — single static binary, Dockerfile, and `docker compose`.

## Architecture

```
Your Services (Go)
   │  import WatchTower SDK — catch panics/errors, batch, send over HTTP
   ▼
Ingestion API (WatchTower backend)
   │  validate API key, fingerprint, deduplicate into issues, store
   ▼
Postgres (issues + events + alert rules + recipients)
   │
   ├──► Notification worker (throttled email alerts)
   └──► Dashboard (same backend, embedded frontend)
```

## Quickstart (Docker Compose)

Prereqs: [Docker](https://docs.docker.com/) with Compose v2.

```bash
git clone <repo> && cd WatchTower
docker compose up -d --build        # or: make docker-compose-up
```

- Dashboard: http://localhost:8080
- Default login: `admin` / `superSecret123!` (override with `ADMIN_USERNAME` / `ADMIN_PASSWORD`)

```bash
docker compose logs -f watchtower   # or: make docker-compose-logs
docker compose down                 # or: make docker-compose-down   (volumes preserved)
```

The `postgres` service stays internal to the Compose network; the backend
waits for its healthcheck. Logs persist in the `wt-logs` volume.

Overrides come from a `.env` at the repo root or your shell environment:
`DB_USER`, `DB_PASSWORD`, `DB_NAME`, `ADMIN_USERNAME`, `ADMIN_PASSWORD`,
`JWT_SECRET_KEY`, `JWT_REFRESH_SECRET_KEY`, `WATCHTOWER_PORT`.

## Local Development

Prereqs: Go 1.26+, Node 22+, a running Postgres (16+).

```bash
# 1. Create the database
createdb watchtower        # or: CREATE DATABASE watchtower;

# 2. Configure the backend (backend/.env is loaded at startup)
cp backend/.env.example backend/.env  # adjust DB_* / ADMIN_PASSWORD

# 3. Run backend (:8080, embeds the built UI) + Vite dev server (:5173) together
make dev-all
```

- `make dev` — build the SPA, embed it, run the backend alone.
- `make dev-web` — Vite dev server with hot reload (proxies `/api` to :8080).
- `make build` — production single binary (`bin/watchtower`) with the UI embedded.

> The SPA is embedded at compile time, so after rebuilding the frontend you
> must restart the backend to serve the new bundle.

## Configuration

All settings are environment variables (loaded from `backend/.env` by the
backend, or the container env under Docker).

### Backend

| Variable | Default | Description |
| --- | --- | --- |
| `SERVER_PORT` | `8080` | HTTP listen port |
| `GIN_MODE` | `release` | `debug` or `release` |
| `TZ` | `UTC` | Server timezone |
| `DB_HOST` | `localhost` | Postgres host |
| `DB_PORT` | `5432` | Postgres port |
| `DB_USER` | `postgres` | Postgres user |
| `DB_PASSWORD` | `postgres` | Postgres password |
| `DB_NAME` | `watchtower` | Database name |
| `DB_SSL_MODE` | `disable` | Postgres SSL mode |
| `DB_MAX_OPEN_CONNS` | `25` | Max open connections |
| `DB_MAX_IDLE_CONNS` | `25` | Max idle connections |
| `DB_CONN_MAX_LIFETIME` | `5m` | Connection max lifetime |
| `ADMIN_USERNAME` | `admin` | Default admin user (auto-created on boot) |
| `ADMIN_PASSWORD` | `admin` | Default admin password |
| `JWT_SECRET_KEY` | `watchtower-secret-key` | Access-token signing secret |
| `JWT_EXPIRATION_DURATION` | `24h` | Access-token lifetime |
| `JWT_REFRESH_SECRET_KEY` | `watchtower-refresh-secret-key` | Refresh-token signing secret |
| `JWT_REFRESH_EXPIRATION_DURATION` | `168h` | Refresh-token lifetime |
| `RATE_LIMIT_REQUESTS` | `100` | Requests per window per client |
| `RATE_LIMIT_WINDOW` | `1m` | Rate-limit window |
| `CORS_ALLOWED_ORIGINS` | `*` | Comma-separated origins |
| `CORS_ALLOWED_METHODS` | `GET,POST,PUT,PATCH,DELETE,OPTIONS` | Allowed methods |
| `CORS_ALLOWED_HEADERS` | `Origin,Content-Type,Accept,Authorization,X-Request-ID,X-Api-Key` | Allowed headers |
| `LOGGER_OUTPUT` | `stdout` | `stdout`, `file`, or `both` |
| `LOGGER_LEVEL` | `info` | Log level (`debug`, `info`, `warn`, `error`, ...) |
| `LOGGER_FILENAME` | `watchtower` | Log file prefix |
| `LOGGER_LOG_DIR` | `logs` | Log directory |
| `LOGGER_MAX_SIZE_MB` | `100` | Max log file size before rotation |
| `LOGGER_MAX_BACKUPS` | `7` | Rotated files kept |
| `LOGGER_MAX_AGE_DAYS` | `30` | Days to keep rotated files |
| `LOGGER_COMPRESS` | `false` | Compress rotated files |
| `LOGGER_ENABLE_CALLER` | `false` | Include caller in log lines |

### Self-reporting (dogfooding)

| Variable | Default | Description |
| --- | --- | --- |
| `SELF_REPORT_ENABLED` | `true` | Report the backend's own errors to itself |
| `SELF_REPORT_PROJECT` | `watchtower-self` | Project label for self-reports |
| `SELF_REPORT_RELEASE` | *(empty)* | Version/commit attached to self-reports |
| `SELF_REPORT_LEVEL` | `fatal` | Minimum forwarded log level (`error`, `fatal`, ...) |

Self-reports are delivered **in-process** — no API key or endpoint is required.
The SDK client is given a `Sender` that hands events straight to the local
ingestion controller, so the full dedup/regression/alert pipeline still runs.
Set `SELF_REPORT_ENABLED=false` to turn it off.

### Go SDK (`ConfigFromEnv`)

| Variable | Description |
| --- | --- |
| `WATCHTOWER_BASE_URL` | WatchTower instance, e.g. `http://localhost:8080` |
| `WATCHTOWER_API_KEY` | API key for the reporting project |
| `WATCHTOWER_PROJECT` | Project label |
| `WATCHTOWER_TAG` | Default tag applied to every report |
| `WATCHTOWER_RELEASE` | Version/commit attached to every report |
| `WATCHTOWER_SAMPLE_RATE` | Sampling in `[0,1]` (default `1.0`) |

## Go SDK

A published, zero-dependency Go module. Install it with:

```bash
go get github.com/watch-tower-org/watchdog/sdk/go@latest
```

Import the module and initialize a process-wide client:

```go
import wt "github.com/watch-tower-org/watchdog/sdk/go"

func main() {
    if err := wt.Init(wt.Config{
        BaseURL: "http://localhost:8080",
        APIKey:  "wt_...",      // create a key in the dashboard
        Project: "payments-api",
        Tag:     "production",
    }); err != nil {
        log.Fatal(err)
    }
    defer wt.Close() // flushes buffered events on shutdown
}
```

Report errors asynchronously (never blocks, never panics):

```go
wt.Report(err,
    wt.WithTag("checkout"),
    wt.WithContext(map[string]any{"order_id": "ord_1234", "attempt": 3}),
)
```

Report synchronously and get the backend result (tests, low-volume paths):

```go
results, err := wt.ReportSync(err)
// results[0].IssueID, results[0].IsNewIssue, results[0].WasRegression ...
```

Recover panics in an `net/http` server — reports and returns 500 without
killing the process:

```go
mux := http.NewServeMux()
// ...
http.ListenAndServe(":8080", wt.RecoverMiddleware()(mux))
```

Wrap goroutine bodies so panics are reported *and* re-raised (Sentry-style
`RecoverHandler`), e.g. for NATS or worker subscribers.

Other knobs:

- `wt.DefaultConfig()` returns sane defaults (5s flush interval, batch size
  100, queue 5000, 10s HTTP timeout).
- `wt.ConfigFromEnv()` builds a config from `WATCHTOWER_*` variables.
- `WithTimestamp`, `WithErrorType` round out the report options.
- `Config.Sender` overrides delivery: implement `Send(ctx, []Event)` and
  events never touch HTTP (`BaseURL`/`APIKey` become optional). The
  self-reporting backend uses this to post events straight into its local
  ingestion pipeline (see below).

A runnable example lives in [`sdk/go/example`](sdk/go/example/main.go).

## Self-reporting (dogfooding)

The backend uses the very SDK it ships, but with zero setup: no API key, no
`.env` editing. The `wt.Client` is configured with a custom `Sender` that
hands captured events directly to the local ingestion controller, so
self-reports flow through the same fingerprint/dedup/regression/alerting
pipeline as any SDK event — they land under the `watchtower-self` project.

Sources of self-reports:

- **Recovered panics** (gin recovery middleware), tagged `panic.recovered`,
  with `url` / `method` / `client_ip` context.
- **Logs** at or above `SELF_REPORT_LEVEL`, tagged `log.<level>`. `fatal` is
  quiet; `error` also forwards 5xx request logs and other logged errors.
  Fatal/Panic logs are reported synchronously before the process exits.

If Postgres itself is down, self-reports can't persist — the sink drops them
and logs a single failed-ingest line while the regular stdout/file log output
keeps working, so visibility is never lost exactly when things break hardest.
A reentrancy guard stops the failure from re-triggering self-reports in a
loop.

## API Overview

All routes are under `/api/watchtower/v1`. Ingestion uses **API-key** auth
(`X-Api-Key` or `Authorization: Bearer`); everything else uses JWT bearer
tokens from `/auth/login`.

| Group | Methods | Purpose |
| --- | --- | --- |
| `/auth` | `POST /login`, `POST /refresh-token`, `POST /logout` | Admin auth |
| `/events` | `POST` (ingest) | SDK reporting — single object or array |
| `/issues` | `GET` (list), `GET /:id`, `PUT /:id` | Issue list/detail/status |
| `/events` | `GET` (list), `GET /:id` | Event history/detail |
| `/dashboard` | `GET /summary` | Dashboard stats |
| `/alert-rules` | `GET`, `POST`, `GET /:id`, `PUT /:id`, `DELETE /:id` | Alert rule CRUD |
| `/alerts` | `GET` (list), `GET /:id` | Alert history |
| `/api-keys` | `GET`, `POST`, `GET /:id`, `PUT /:id`, `PUT /:id/revoke` | API key management |
| `/recipient-lists` | `GET`, `POST`, `GET /:id`, `PUT /:id`, `DELETE /:id` | Alert recipients |
| `/settings` | `GET`, `PUT` | General settings |
| `/settings/email` | `GET`, `PUT`, `POST /test` | SMTP settings + test email |
| `/settings/alert` | `GET`, `PUT` | Global alert/throttle settings |

Ingestion payload keys: `message`, `error_type`, `stack_trace`, `project`,
`tag`, `context`, `timestamp`. The response echoes `{received, events[]}` with
per-event `issue_id`, `event_id`, `is_new_issue`, `was_regression`,
`fingerprint`.

## Project Layout

```
sdk/go/     Go SDK (stdlib-only): capture, transport, batching, middleware
backend/    Go backend: cmd entrypoint (main.go), internal/ (config, db,
            logger, middleware, model), pkg/ (feature controllers + routers),
            web/ (embedded SPA dist)
web/        React + Vite dashboard
docker/     Multi-stage Dockerfile (web -> go -> scratch)
docs/       docs.md (full spec), kuwait-university-reference.md
```

## Releasing the SDK

The SDK is a nested Go module (`github.com/watch-tower-org/watchdog/sdk/go`).
Release a new version with a submodule tag:

```bash
git tag sdk/go/v0.1.1          # tag shape matters: <module dir>/v<version>
git push origin sdk/go/v0.1.1  # consumers then: go get .../sdk/go@v0.1.1
```

The backend depends on the SDK via `replace ../sdk/go` for local development, so
backend and SDK always build together regardless of the last published tag.

## Testing

```bash
cd sdk/go && go vet ./... && go test ./...
cd backend && go vet ./... && go test ./...
npm --prefix web run build     # frontend typecheck + build
```

## Documentation

See [`docs/docs.md`](docs/docs.md) for the full design spec (architecture,
SDK contract, ingestion pipeline, alerting, self-reporting).
