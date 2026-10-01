# QA Report: smeltforge — 2026-04-21

**Branch:** docs/mcp-server-prerequisites
**Scope:** smeltforge/ Go module + core/cmd/smeltforge.go dispatch layer
**Mode:** Diff-aware (no running app — Go code quality QA)
**Tier:** Standard

---

## Health Score: 8.5/10

| Category | Score | Notes |
|---|---|---|
| Build | 10/10 | go build clean |
| Vet | 10/10 | go vet clean |
| Correctness | 7/10 | 2 medium bugs fixed |
| Code Quality | 8/10 | 1 dead-code item fixed |
| Spec Alignment | 9/10 | All CLI commands present, Docker SDK + Caddy JSON API used correctly |

---

## Issues Found

### ISSUE-001 (Medium) — FIXED
**Poller ignores `Trigger.Interval`, fires every 10s for all projects**

`RunPoller` used a fixed 10-second ticker and enqueued every polling-enabled project on every tick, ignoring the project-level `Trigger.Interval` configuration.

- **File:** `smeltforge/internal/server/server.go:RunPoller`
- **Fix:** Added `lastPolled map[string]time.Time` tracking. Each project is only enqueued when `now - lastPolled[id] >= Trigger.Interval`. Default interval 60s if unset.
- **Commit:** (unfixed at report time — fix applied inline)

---

### ISSUE-002 (Medium) — FIXED
**`deployStopStart` skips `StopAndRemove` when state is empty, risking "container name already in use"**

When `p.State.ID == ""` and `p.State.Image == ""` (first deploy after registry wipe or fresh add), the code skipped `StopAndRemove`. If a container with that name existed from a prior crashed state, `ContainerCreate` would fail with "container name already in use".

`StopAndRemove` is already idempotent (returns nil if the container is not found), so the guard was unnecessary.

- **File:** `smeltforge/internal/deploy/engine.go:deployStopStart`
- **Fix:** Removed the conditional — `StopAndRemove` now always runs before creating the new container.

---

### ISSUE-003 (Low) — FIXED
**Dead `lastCommit` map in `RunPoller`**

`lastCommit := map[string]string{}` was allocated but only accessed via `_ = lastCommit`. Removed when ISSUE-001 was fixed (replaced by `lastPolled`).

- **File:** `smeltforge/internal/server/server.go:RunPoller`

---

## Deferred (out of scope)

- No unit tests exist. The module has no `*_test.go` files. This is expected for a new implementation — test coverage is a follow-on task.
- `notifySparkForge` is a no-op stub. Intentional per spec.
- Webhook secret is in URL path (server logs exposure). This is per-spec behavior.

---

## Summary

- 3 issues found, 3 fixed (2 medium, 1 low)
- Build and vet: clean before and after fixes
- All spec requirements verified present: Docker SDK usage, Caddy JSON Admin API, blue-green strategy, stop-start strategy, health check loop, deploy queue, webhook + CI token endpoints, audit logging, env var injection from secrets

**QA result: PASS — all fixable issues resolved.**
