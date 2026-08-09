# WatchTower TODO

Improvements / fixes / additions for the self-hosted error tracking & alerting
system. Worked through top-to-bottom, one item at a time.

Legend: `[priority: high]` = do first · `med` = next · `low` = nice to have.

## Confirmed Bugs

- [x] [priority: high] Dashboard 404 — `DashboardPage.tsx:40` calls `GET /settings/alerts` (plural); backend only registers `/settings/alert` (singular). "Alert throttling" card always shows `0 min`. Fix the frontend path.
- [x] [priority: high] `error_type` dropped in dedup — `backend/pkg/ingestion/controller.go:29` calls `ExtractErrorType(req.Message, req.StackTrace)` and never reads `req.ErrorType`; SDK's `WithErrorType` has no effect on grouping. Worse, `ExtractErrorType` (`fingerprint.go:25`) returns the whole message when it has no colon, so `"user 123 not found"` vs `"user 456 not found"` on the same line become different issues. Prefer the client-supplied error type; only fall back to message/stack heuristics. *(Fixed: `ResolveErrorType` prefers `req.ErrorType`; colon-less messages no longer used as type.)*
- [x] [priority: high] Rate-limit header bug — `backend/internal/middleware/rate_limit.go:38` sets `X-RateLimit-Limit` from the *request* header (`c.GetHeader`) instead of the actual limit. Set the real configured limit + remaining/reset. *(Fixed: `X-RateLimit-Limit` = configured limit, added `X-RateLimit-Remaining`, `X-RateLimit-Reset` = epoch seconds.)*
- [x] [priority: high] Notifier workers param ignored — `backend/pkg/notifier/notifier.go:52` accepts `workers` but line 69 always starts `defaultWorkers` (5). Use the passed value. *(Fixed: stored in struct + used by `Start`; exposed as `NOTIFIER_WORKERS` env, wired in `app.go`.)*

## Dedup / Grouping Quality

- [x] [priority: med] Message-independent fingerprinting — fingerprint on (project, error_type, normalized stack frames) where error_type is the stable Go type from the SDK, not message text. Consider a configurable fingerprint mode (`message` / `type` / `type+frames`). *(Done: default is type+frames; `FINGERPRINT_MODE` env adds `type` and `message` modes via `ComputeFingerprintMode`/`FingerprintMode` in `fingerprint.go`.)*
- [x] [priority: med] Merge / split issues in the dashboard — admin can move events between issues or split a noisy issue (escape hatch for imperfect dedup). *(Done: `POST /issues/merge` (move events into a kept target, recompute aggregates, delete sources) and `POST /issues/move-events` (reparent to an existing issue or split into a new issue with a random fingerprint so new events never auto-land in it); dashboard merge UI + per-event move/split UI.)*
- [x] [priority: low] Sentry-style grouping heuristics — group by error type + top frame function + normalized message; expose the fingerprint in the UI for debugging. *(Done: new opt-in `FINGERPRINT_MODE=heuristic` — masks ids/timestamps/urls/ips/hex/numbers in the message and groups by error type + top frame function via `ExtractFrameFunction`; fingerprint shown on the issue detail page and as a tooltip on the issues list.)*

## Alerting

- [ ] [priority: high] Additional notification channels — Slack / Discord / generic webhook / PagerDuty in addition to email. Biggest missing feature for a monitoring tool.
- [ ] [priority: med] New trigger types — `daily_report` (daily digest of top issues), `noisy_issue` (auto-mute when exceeding a threshold), `snooze` / `silence_until` on issues.
- [ ] [priority: med] Auto-resolution — mark an issue "resolved" after N days with no events so regression tracking works over time.
- [x] [priority: low] Throttle semantics cleanup — unset `throttle_window` stores `0` (frontend `AlertRulesPage.tsx:167` misleadingly renders "global min"); document or use a nullable column with the global default applied at eval time. Add `events(timestamp)` index so spike `COUNT` queries don't table-scan per rule per event. *(Done: `AlertRule.throttle_window` is now nullable — `null`/`0` = "use global setting", applied at eval time in `notifier/evaluate.go`; migration relaxes NOT NULL and normalizes stored `0`s; frontend shows the effective value `global (N min)` and sends `null` to clear; `events(timestamp)` index added.)*

## Reliability / Scaling

- [x] [priority: med] `events(timestamp)` index — spike queries and any retention job currently scan an unbounded table. *(Done: index added in the throttle-migration change; the retention job will rely on it.)*
- [x] [priority: med] Clamp `page_size` — no upper bound anywhere (`?page_size=1000000` is honored); add a max (e.g. 100). *(Done: shared `pagination.Normalize` clamps `page_size` to 1–100 across all list endpoints.)*
- [ ] [priority: med] Idempotent ingestion — `IngestBatch` commits events one-by-one then returns error mid-way; SDK retries the whole batch and inflates `issues.count`. Add a per-batch dedup/idempotency key or per-event error reporting.
- [ ] [priority: med] Bounded goroutine spawns — per-API-key `last_used_at` update runs an unbounded goroutine per request under ingest load; move onto the worker pool.
- [ ] [priority: low] Versioned migrations — currently `CREATE TABLE IF NOT EXISTS` + ad-hoc `ALTER TABLE ADD COLUMN IF NOT EXISTS` in `backend/internal/database/migration.go`; drift-prone. Move to `golang-migrate` (or a `schema_migrations` table) with ordered, reversible migrations.
- [ ] [priority: low] DB-backed / shared rate limiting — current limiter is per-process in-memory keyed by IP (breaks behind proxies, not shared across instances); consider skipping the SPA and ingestion paths.

## Security

- [ ] [priority: med] Token revocation — `POST /auth/logout` is a no-op and refresh-token doesn't re-check the admin exists/active (`backend/pkg/auth`). Add a blocklist or rotate refresh tokens; validate admin status on refresh.
- [ ] [priority: med] Default-secret foot-gun — config defaults `ADMIN_PASSWORD=admin`, `JWT_SECRET_KEY=watchtower-secret-key` when env absent; fail fast (or warn loudly) when running in production mode with defaults.
- [x] [priority: low] Harden SPA auth — move tokens from `localStorage` (XSS-accessible) to httpOnly cookies; add CSP, HSTS, Referrer-Policy to `SecurityHeaders`. *(Done: `/auth/login` sets httpOnly `wt_access_token`/`wt_refresh_token` cookies (SameSite=Lax, `Secure` when `COOKIE_SECURE=true`, disabled by default); token refresh rides the cookie; Bearer header still supported for API clients; CSP/HSTS/Referrer-Policy added; `/auth/me` drives SPA bootstrap.)*

## Frontend

- [ ] [priority: med] Tests + CI — zero frontend tests and no CI on either side. Add vitest/RTL for pages + utils and a `.github/workflows` gate (oxlint, typecheck, build, `go vet`, `go test`).
- [x] [priority: med] Live dashboard — `refetchOnWindowFocus: false` and no `refetchInterval` means counts go stale; add auto-refresh (it's a monitoring tool). *(Done: dashboard summary refetches every 15s + on window focus.)*
- [x] [priority: low] Fix render-time anti-patterns — `setForm` during render in `SettingsPage.tsx`, `navigate` during render in `SetupPage.tsx`; extract shared `PaginatedTable` (pagination markup copy-pasted across 6 pages); debounce search inputs. *(Done: both anti-patterns moved into `useEffect`; shared `PaginationControls` component replaces the copy-pasted blocks on all 6 pages; search/filter inputs debounced via `useDebouncedValue`.)*
- [x] [priority: low] Delete dead code — `App.css` (never imported), unused `assets/*` + `public/icons.svg`, unused `next-themes` dep, unused `components/ui/{separator,skeleton}.tsx`; README is boilerplate Vite template. *(Done: all deleted; `next-themes` removed from deps; boilerplate `web/README.md` removed.)*

## Ops / Observability

- [ ] [priority: med] Retention / TTL job — prune old events/issues (docs.md lists as an open question); pair with the `events(timestamp)` index.
- [ ] [priority: med] `/metrics` (Prometheus) + `/health` / `/ready` endpoints — currently only `/version` exists; a monitoring tool should be self-monitorable.
- [ ] [priority: low] Repo hygiene — `backend/logs/*.log` (incl. `self-report.key`) committed to git; `.DS_Store` present; `docs/docs.md` §7 still documents `/api/events` instead of `/api/watchtower/v1/events`.
- [ ] [priority: low] Embedded-build foot-gun — fresh-clone `go build` serves the placeholder "frontend not built" page; fail the build or CI-check that `web/dist` exists.

## New Features

- [ ] [priority: med] SDK breadcrumbs / performance spans — capture HTTP server requests, DB calls, timings as breadcrumbs attached to error reports; today only errors + panics are captured.
- [ ] [priority: low] Full-text search + saved views — search across issues/events, time-range filters on the dashboard, saved filter presets.
- [ ] [priority: low] Other-language SDKs — Python / Node / TypeScript client once the Go path and API contract are stable.
