# WatchForge QA Report

**Date:** 2026-04-22
**Module:** watchforge/
**Scope:** Full implementation review — all packages, all CLI commands, daemon lifecycle

---

## Build & Vet

| Check | Result |
|-------|--------|
| `go build ./watchforge/...` | PASS |
| `go build ./core/...` | PASS |
| `go build ./shared/...` | PASS |
| `go vet ./watchforge/... ./core/... ./shared/...` | PASS |

---

## Tests Written and Passing

| Package | Tests | Result |
|---------|-------|--------|
| `internal/checker` | 12 | PASS |
| `internal/registry` | 10 | PASS |
| `internal/scheduler` | 6 | PASS |
| `internal/api` | 9 | PASS |
| `internal/statuspage` | 3 | PASS |
| **Total** | **40** | **ALL PASS** |

---

## Smoke Tests: CLI Commands

All 9 commands tested against a live binary:

| Command | Test | Result |
|---------|------|--------|
| `watchforge init` | Creates dirs + empty registry files | PASS |
| `watchforge add --type http` | Creates monitor, returns ID | PASS |
| `watchforge add --type tcp` | Creates monitor | PASS |
| `watchforge add --type docker` | Creates monitor | PASS |
| `watchforge add --type ssl` | Creates monitor | PASS |
| `watchforge add --type heartbeat` | Creates monitor + prints heartbeat URL | PASS |
| `watchforge list` | Tabular output; JSON via `--output json` | PASS |
| `watchforge status` | HEALTHY/DOWN/PAUSED derived from incidents | PASS |
| `watchforge pause` | Sets paused=true; idempotent | PASS |
| `watchforge resume` | Sets paused=false; idempotent | PASS |
| `watchforge delete --yes` | Removes monitor atomically | PASS |
| `watchforge incidents` | Lists incidents; `--since Nd` filter | PASS |
| `watchforge update` | Partial field updates via Flags().Changed() | PASS |
| `watchforge heartbeat-url` | Returns correct URL; fails on non-heartbeat type | PASS |
| `--output json` on all commands | Valid JSON, non-zero exit on error | PASS |
| Error paths (missing IDs, bad types) | Non-zero exit + error message | PASS |

---

## Smoke Tests: Daemon + API

| Test | Result |
|------|--------|
| Daemon starts and logs version + API addr | PASS |
| Daemon stops cleanly on SIGTERM | PASS |
| `POST /v1/watchforge/pause?monitor=<id>` | Pauses monitor, returns JSON | PASS |
| `POST /v1/watchforge/resume?monitor=<id>` | Resumes monitor, returns JSON | PASS |
| `GET /_watchforge/heartbeat/<id>/<token>` | Records ping, updates LastPing | PASS |
| Wrong token -> 403 Forbidden | PASS |
| Non-heartbeat monitor -> 400 Bad Request | PASS |
| Missing `monitor` param -> 400 | PASS |
| GET on pause/resume -> 405 Method Not Allowed | PASS |

---

## Bug Found and Fixed

**`core/cmd/watchforge.go` — ID/token generation used `/dev/urandom` directly**

The CLI used `os.Open("/dev/urandom")` with silent error swallowing. If the file
failed to open, all IDs would be `00000000-0000-0000-0000-000000000000`.

Fix: switched to `crypto/rand.Read()`, matching the daemon's `scheduler.newID()`.

---

## Behavioral Verification

| Behavior | Verified |
|----------|---------|
| Incident opens after threshold=2 consecutive failures | YES |
| No duplicate incidents (deduplication via activeIncidentID guard) | YES |
| consecutiveFails resets to 0 on recovery | YES |
| Open incidents restored from incidents.json on daemon restart | YES |
| Private monitors hidden from status page | YES |
| Atomic writes (temp + rename) for monitors.json, incidents.json, index.html | YES |
| Heartbeat URL format: `https://status.<domain>/_watchforge/<id>/<token>` | YES |
| Status page only shows public monitors | YES |
| 30-day uptime calculation | YES |
| 90-day incident history (closed only) | YES |

---

## Health Score: 9/10

**Reason for -1:** No integration test covering a full check-cycle (run checker -> open incident -> recover -> close incident) with the real scheduler goroutine. The unit tests cover all pieces in isolation; the scheduler integration test covers state restoration. A goroutine-level integration test would require either a mock checker or a real local server, adding test complexity without blocking ship readiness.

**Ship readiness: YES** — all implemented requirements verified, no open bugs.

