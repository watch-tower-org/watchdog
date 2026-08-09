# WatchTower — Self-Hosted Error Tracking System

## 1. Overview

WatchTower is a self-hosted error tracking and alerting system built for
microservices projects (Go backends, NATS-based services, Postgres-backed
apps, etc). It provides SDK-based error capture, deduplication, email
alerting, and a dashboard — without the overhead of multi-project /
multi-tenant management. One self-hosted instance = one team, all their
services report into it.

### Goals
- Simple self-hosting via Docker (single image preferred).
- SDK-based integration (no external transport like NATS/Kafka required).
- Deduplication of repeated errors into "issues".
- Configurable email alerting with throttling (avoid alert spam).
- Dashboard for viewing events, issues, and alert history.
- No project-management layer in v1 — projects are just a label set at
  SDK init time, not a managed entity with its own permissions/settings.

### Non-goals (v1)
- Multi-tenant / multi-org support.
- Per-project API keys with a full projects CRUD UI (see open question below).
- Distributed tracing / cross-service correlation.
- Non-Go SDKs (initial release is Go-only; other languages later).

---

## 2. Architecture

```
Your Services (Go, etc.)
   │  import WatchTower SDK
   │  SDK catches panics/errors, batches, sends over HTTP
   ▼
Ingestion API (WatchTower backend)
   │  validates API key, deduplicates, groups into issues, stores
   ▼
Postgres (events + issues + alert rules + recipients)
   │
   ├──► Notification worker (throttled email alerts)
   └──► Dashboard (served by the same backend, embedded frontend)
```

### Components
1. **SDK** (Go package, imported by services)
2. **Backend / Ingestion API** (Go service — HTTP routers, Postgres
   connection, self-error-reporting)
3. **Backoffice / Dashboard** (frontend, likely embedded into the backend
   binary and served as static files — single Docker image)
4. **Postgres** (storage)

### Deployment shape
- **Preferred: single Docker image.** Go backend embeds the built
  frontend (via `embed.FS`) and serves both the ingestion API and the
  dashboard UI from one binary/container.
- `docker-compose.yml` for self-hosters: `app` + `postgres` (+ optional
  `mailhog`/SMTP relay for local dev/testing).
- Config via environment variables and/or a first-boot setup flow.

---

## 3. Configuration Flow (first boot)

1. **Email/SMTP settings** — host, port, credentials, from-address.
2. **API key** — generated once, shown to the user, used by all SDK
   instances across all services.
3. **Recipient email list(s)** — one or more named lists of email
   addresses that alert rules can target.
4. **Alert throttling settings** — e.g. "don't re-alert on the same issue
   within N minutes," configurable globally (per-rule override possible
   later).

After setup, the dashboard becomes live with three main views:

- **Events** — raw reported error occurrences.
- **Issues** — deduplicated/grouped errors (one issue = one fingerprint).
- **Alerts** — history of what fired, when, and to whom.

---

## 4. Data Model (rough shape)

- `issues`
  - `id`, `fingerprint` (hash of service + error type + top N stack
    frames), `title`, `project`, `tag`, `status` (open/resolved/muted),
    `first_seen`, `last_seen`, `count`
- `events`
  - `id`, `issue_id`, `timestamp`, `stack_trace`, `context` (JSON blob),
    `project`, `tag`
  - Consider capping stored raw events per issue (e.g. last 50) plus a
    running counter, to avoid unbounded storage growth on hot-loop bugs.
- `alert_rules`
  - `id`, `trigger_type` (new issue / spike / regression), `matcher`
    (project/tag filter), `throttle_window`, `recipient_list_id`
- `recipient_lists`
  - `id`, `name`, `emails[]`
- `alert_log`
  - `id`, `issue_id`, `rule_id`, `sent_at`, `recipients[]`

---

## 5. Deduplication & Alert Throttling

Two separate mechanisms, working together:

1. **Fingerprint-based deduplication** (storage level)
   - Incoming events are hashed by `(project, error type, top N stack
     frames)`.
   - Matching fingerprint → increments existing issue's `count` and
     updates `last_seen` instead of creating a new issue.

2. **Alert throttling** (notification level)
   - Even though every occurrence updates the issue, alert rules only
     fire once per throttle window per issue (e.g. once per hour), even
     if the error happens thousands of times in that window.
   - This is the core mechanism for "avoid sending multiple emails for
     the same error over and over."

### Alert triggers
- New issue created (first time this fingerprint is seen).
- Issue exceeds N occurrences within T minutes (spike detection).
- A previously resolved issue reoccurs (regression).

### Recipients
- Alert rules point to one or more recipient lists (many-to-many).
- Recipient lists are reusable across rules (e.g. "backend-team",
  "oncall").

---

## 6. SDK Design (Go)

### Initialization
```go
wt.Init(wt.Config{
    BaseURL: "http://localhost:8080",
    APIKey:  "wt_xxx",
    Project: "backend-api", // set once at init, labels all reports from this service
})
```

### Manual reporting
```go
wt.Report(err, wt.WithTag("payment"), wt.WithContext(map[string]any{
    "user_id":  userID,
    "order_id": orderID,
}))
```

### Automatic panic recovery
```go
router.Use(wt.RecoverMiddleware())
```
- Should also wrap NATS subscriber handlers (or expose a similar
  `wt.RecoverHandler(fn)` wrapper) so panics inside async consumers are
  captured, even though NATS isn't used as the transport for error
  reporting itself.

### Behavior
- Attaches automatically: project (from Init), service hostname, git
  commit/version (if provided), stack trace, timestamp.
- Attaches manually: tag, arbitrary key/value context.
- **Batches events client-side** (flush every N seconds or M events)
  rather than firing one HTTP call per error — avoids adding latency or
  becoming a cascading-failure source during an incident.
- Config via `Init()` struct and/or environment variables (base URL, API
  key, project, sample rate).

---

## 7. Backend / Ingestion API

- `POST /api/events` — SDK reporting endpoint, authenticated via API key.
- Admin/dashboard routes for issues, events, alert rules, recipient
  lists, settings.
- Connects to Postgres for storage.
- **Self-reports its own errors** using the same SDK/mechanism it exposes
  to other services (dogfooding).
  - Implemented by `backend/pkg/selfreport`: the backend holds a `wt.Client`
    configured with a custom `Sender` that hands captured events directly to
    the local ingestion controller, so no API key or HTTP endpoint is
    required. Self-reports flow through the real fingerprint/dedup/alert
    pipeline and land under the `watchtower-self` project.
  - Sources: recovered panics (gin recovery middleware reports with
    url/method/client_ip context) and zerolog messages at or above
    `SELF_REPORT_LEVEL` (default `fatal`; `error` forwards 5xx request logs
    and other logged errors, tagged `log.<level>`). Fatal/Panic logs are
    reported synchronously because zerolog exits the process right after
    writing them.
  - Important edge case: if the backend's own error is caused by
    Postgres being unavailable, self-reporting must not depend on a
    working DB write. Fall back to stderr/log file in that case so
    visibility isn't lost during the exact moment things break hardest.
    The sink logs a single failed-ingest line while the DB is down, and a
    reentrancy guard (the log hook is suppressed while the ingest itself is
    running) prevents the failure from re-triggering self-reports in a loop.
    The zerolog stdout/file output still works.

---

## 8. Open Questions

1. **API key scope**: one global API key for the whole instance (simplest,
   `project` is just a label in the payload), vs. one key per project
   even without a full projects UI (keys as config-file entries, not a
   managed DB entity) — gives a cheap security boundary so a leaked key
   only affects one service. **Decision needed before SDK auth is
   finalized.**
2. Per-rule throttle override vs. global-only throttle for v1.
3. Retention policy specifics (event TTL, sampling thresholds) — revisit
   once real volume is observed.

---

## 9. Suggested Build Order

1. Postgres schema + Ingestion API (`POST /api/events`) + fingerprinting
   and dedup logic.
2. Go SDK: `Init`, `Report`, `RecoverMiddleware`, client-side batching.
3. Email notifier with throttled alert rules.
4. Dashboard (issues list/detail, events, alerts, settings) — embedded
   into the backend binary.
5. First-boot setup flow (SMTP, API key, recipient lists, throttle
   config).
6. Retention/sampling once volume becomes a real concern.

---

## 10. Self-Hosting with Docker

A single multi-stage `Dockerfile` builds the web dashboard and embeds it
into a static Go binary (`scratch` runtime, no shell). The image is published
on Docker Hub as `watchtowerorg/watchtower` (`latest` / `vX.Y.Z`,
linux/amd64 + linux/arm64). `docker-compose.yml` at the repo root runs the
full stack, pulling the published image by default:

```bash
docker compose up -d             # pulls watchtowerorg/watchtower (or: make docker-compose-up)
# dashboard: http://localhost:8080  (admin / ADMIN_PASSWORD, default superSecret123!, set on first boot only)
docker compose logs -f watchtower
docker compose down              # volumes (DB, logs) are preserved
```

- To build from source instead of pulling: `docker compose up -d --build`
  (compose builds from `docker/Dockerfile` and tags it as
  `watchtowerorg/watchtower:latest` locally); `make docker` builds a separate
  local image named `watchtower:local`.
- Publish a new multi-arch release with `make docker-publish VERSION=vX.Y.Z`
  (requires a `multiarch` buildx builder:
  `docker buildx create --name multiarch --driver docker-container --bootstrap`).
- `postgres` service is internal (not published to the host); it runs a
  healthcheck and the backend waits for it.
- Self-reporting works in the container too — it is in-process, so no extra
  configuration is needed; logs persist in the `wt-logs` volume.
- Overridable via a `.env` at the repo root (or shell env): `DB_USER`,
  `DB_PASSWORD`, `DB_NAME`, `ADMIN_USERNAME`, `ADMIN_PASSWORD`,
  `ADMIN_RESET_PASSWORD`, `JWT_SECRET_KEY`, `JWT_REFRESH_SECRET_KEY`,
  `WATCHTOWER_PORT`. The admin password is applied on first boot only;
  set `ADMIN_RESET_PASSWORD=true` to force a reset on the next boot.
- The backend `go.mod` replaces the SDK with a local path (`../sdk/go`);
  the `Dockerfile` copies `sdk/` into the builder so the replacement
  resolves during the image build.

---

## 11. SDK Release Process

The SDK is a nested Go module (`github.com/watch-tower-org/watchtower/sdk/go`,
stdlib-only, no `go.sum`). It is published with **submodule version tags**:
the tag must be `<module dir>/v<version>`, i.e. `sdk/go/v0.1.0`, so Go can
locate the module boundary inside the monorepo.

```bash
# after committing the SDK changes
git tag sdk/go/vX.Y.Z
git push origin sdk/go/vX.Y.Z

# consumers install it with
go get github.com/watch-tower-org/watchtower/sdk/go@vX.Y.Z
```

- The backend pins the SDK with `replace ../sdk/go` (local path), so
  development never waits on a tag; the `require` line mirrors the last
  published version for readability.
- Verify a fresh install with `GOPROXY=direct` (bypasses the module-proxy
  cache, which may lag a brand-new module by a few minutes).
- `sdk/go/example` ships as a runnable example inside the module.
- Hardening notes: the module declares `go 1.22` for wide toolchain
  compatibility; `error_type` is derived from the error's Go type unless
  overridden with `WithErrorType`; `Config.MaxRetries` (default 2) retries
  failed background batch flushes in the flusher goroutine; a `SampleRate`
  of `0` drops every event (callers should build configs from
  `DefaultConfig()`/`ConfigFromEnv()`); re-calling `Init` replaces and
  flushes the previous client.
