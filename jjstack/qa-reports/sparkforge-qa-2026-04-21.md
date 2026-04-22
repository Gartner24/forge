# QA Report: sparkforge -- 2026-04-21

**Branch:** docs/mcp-server-prerequisites
**Scope:** sparkforge/ Go module -- notification orchestration daemon
**Mode:** API/integration testing (no browser; Go CLI + HTTP server)
**Tier:** Standard

---

## Health Score: 9/10

| Category | Score | Notes |
|---|---|---|
| Build | 10/10 | `go build ./sparkforge/...` clean |
| Vet | 10/10 | `go vet ./sparkforge/...` clean |
| Correctness | 10/10 | 55/55 tests pass, 0 failures |
| Coverage breadth | 8/10 | delivery/ package untested in isolation (see deferred) |
| Spec alignment | 9/10 | All CLI commands, auth, dedup, priority routing implemented |

---

## Test Summary

55 tests across 6 packages, all passing.

| Package | Tests | Kano Level |
|---|---|---|
| internal/api | 16 | 1 (Security) |
| internal/router | 14 | 2 (Core) |
| internal/registry | 8 | 2 (Core) |
| internal/dedup | 6 | 2 (Core) |
| internal/deliverylog | 5 | 3 (Auxiliary) |
| internal/model | 5 | 3 (Auxiliary) |

```
ok  github.com/gartner24/forge/sparkforge/internal/api          0.027s
ok  github.com/gartner24/forge/sparkforge/internal/dedup        0.004s
ok  github.com/gartner24/forge/sparkforge/internal/deliverylog  0.004s
ok  github.com/gartner24/forge/sparkforge/internal/model        0.002s
ok  github.com/gartner24/forge/sparkforge/internal/registry     0.006s
ok  github.com/gartner24/forge/sparkforge/internal/router       0.021s
```

---

## Issues Found

No blocking or medium issues found. One minor item and one deferred gap.

### ISSUE-001 (Low) -- Open
**`fmt.Printf` used for non-fatal dedup error in router.go**

`router.go:74` uses `fmt.Printf` to print "sparkforge: failed to record alert: ..." to stdout.
This is a daemon process -- error output should go to stderr.

- **File:** `sparkforge/internal/router/router.go:74`
- **Fix:** Replace `fmt.Printf` with `fmt.Fprintf(os.Stderr, ...)` and add `"os"` import.
- **Severity:** Low -- does not affect correctness, only log destination.

---

## Deferred (out of scope)

**delivery/ package has no direct unit tests**

`internal/delivery/gotify.go`, `email.go`, and `webhook.go` have no `*_test.go` files.

- Gotify and email require real external services (Gotify server, SMTP relay) -- Docker integration tests would be appropriate, deferred to a future QA pass.
- `webhook.go` is fully testable with `httptest.NewServer` and could have unit tests covering: Slack payload format, Discord payload format, generic format, 4xx response handling, empty URL error. The router tests exercise webhook delivery indirectly (all router/channel tests use httptest servers as webhook endpoints), so this is not a correctness gap -- it is a coverage gap.

---

## Correctness Verified

The following correctness-critical behaviors are tested end-to-end:

**Kano 1 -- Security (API auth)**
- No token -> 401
- Wrong token -> 401
- `Token` or raw scheme -> 401 (Bearer prefix required)
- Empty Bearer -> 401
- Cross-namespace token (not under `sparkforge.api_tokens.*`) -> 401
- Valid token -> 200

**Kano 2 -- Core (routing + dedup + registry)**
- Priority filter: below min -> not delivered, at min -> delivered, above min -> delivered
- Disabled channel -> skipped
- Channel A failure does not block channel B (isolation)
- Dedup: same (source, event_type) -> second send no-ops
- Dedup: different event_type -> both delivered
- No event_type -> never deduplicated (all sends go through)
- SendToChannel: targeted delivery, disabled channel returns error
- Delivery log: success written as "ok", failure written as "error" with error field

**Kano 3 -- Auxiliary (delivery log + model)**
- Append is append-only (5 appends -> 5 records)
- Read(since) filters by timestamp
- Error records preserve error string
- Priority.Level() ordering: low < medium < high < critical
- ParsePriority round-trips all valid values
- Invalid priority -> Valid() returns false

---

## Cleanup

All tests used `t.Setenv("HOME", t.TempDir())` for path isolation. No persistent test data created. No cleanup required.
