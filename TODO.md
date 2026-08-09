# WatchTower Improvement Plan

Step-by-step work items derived from a full review of the backend, web dashboard,
and Go SDK. Organized into phases. Tick items off as they land.

- [ ] = pending
- [x] = done

---

## Phase 3 — Dashboard UX

### D1. Charts
- [x] Add chart library (recharts).
- [x] New backend endpoint `GET /dashboard/trends?range=24h|7d|30d` returning per-bucket `{ts, new_issues, events}` (`backend/pkg/dashboard/controller.go`, router, web `types.ts`).
- [x] `web/src/pages/DashboardPage.tsx`: issue-rate area chart + top-projects/top-tags + recent-issues feed (`--chart-*` tokens already in `index.css`).

### D4. Filters & sorting
- [ ] `EventsPage`: message search, tag filter, date-range picker.
- [ ] `AlertsPage`: date range + issue search.
- [ ] `ApiKeysPage`: search box.
- [ ] Sortable columns on Issues/Events tables.

### D5. Consistent loading/error states
- [ ] Shared `Skeleton` component + row skeletons for all tables.
- [ ] Reusable query-error banner with retry (pattern from `IssueDetailPage`).

### D6. Relative timestamps & small wins
- [ ] `timeAgo()` util in `web/src/lib/`; use everywhere timestamps render.

---

## Phase 4 — SDK improvements

### Sdk1. Payload trimming
- [ ] Enforce max lengths on `Message`/`StackTrace` (e.g. 64KB stack), cap `Context` key count + per-value size (`sdk/go/event.go` / `transport.go`).

### Sdk2. Compression
- [ ] gzip the body + `Content-Encoding: gzip` when payload > ~1KB (`sdk/go/transport.go:67`).
- [ ] gzip middleware on the ingestion route (`backend/pkg/ingestion/handler.go`).

### Sdk3. Status-aware retry
- [ ] Don't retry 4xx (except 429); on 429 respect `Retry-After`; add jitter to the 500ms backoff (`sdk/go/batching.go:106-131`).
- [ ] Treat 413 as "trim and resend once" or drop.

### Sdk4. Trace context & request metadata
- [ ] Add `WithRequestID`, `WithUser`, and an `Environment` field to `Config` + `Event` (add `environment` column to backend `model/event.go`).
- [ ] `RecoverMiddleware` (`sdk/go/middleware.go:10-13`): include User-Agent, query string, request ID header.

### Sdk5. Test coverage
- [ ] Tests for `ConfigFromEnv` (currently 0%).
- [ ] Tests for package-level `wt.Report*` / `Close` / `RecoverHandler` (currently 0%).
- [ ] Tests for `transport.Send` error branches (currently ~74%).

---

## Phase 5 — New product features

### F2. Event search
- [ ] `tsvector` full-text index on `events(message, stack_trace)`.
- [ ] `search` param on `GET /events` and `GET /issues`.
- [ ] Frontend search box in `EventsPage` / `IssuesPage`.

### F3. Multi-project / multi-user / RBAC
- [ ] Introduce `projects` table; scope API keys/issues/events/alerts by project.
