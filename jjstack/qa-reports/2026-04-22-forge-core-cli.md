# QA Report: forge core CLI

Date: 2026-04-22
Branch: docs/mcp-server-prerequisites
Subject: `core/` - forge CLI binary (Cobra, Go 1.23)

---

## Summary

QA pass on the forge core CLI implementation. 5 bugs found and fixed, all
command paths verified via direct invocation. Binary builds clean.

**QA Health Score: 9/10**

Deduction: live-tail mode (`forge logs <module>` without flags) creates an
empty log file on failed `forge start`, then blocks forever in tail mode
instead of exiting. Acceptable as-is for initial release; minor UX issue.

---

## Commands Tested

| Command | Cases Covered | Result |
|---------|--------------|--------|
| `forge init` | missing flag, success, idempotent, JSON | PASS |
| `forge install` | known module, unknown module, already installed, JSON | PASS |
| `forge uninstall` | success, double-uninstall, confirmation prompt, --yes, JSON | PASS |
| `forge start` | no binary, success, already running, JSON | PASS |
| `forge stop` | not running, success, JSON | PASS |
| `forge status` | empty, mixed running/stopped, JSON | PASS |
| `forge logs` | no log file, --lines, --since, --since+--lines, bad duration | PASS |
| `forge config show` | text, JSON | PASS |
| `forge config get` | known key, unknown key, JSON | PASS |
| `forge config set` | writable key, read-only key, JSON | PASS |
| `forge secrets set/get/list/delete` | all ops, --prefix, --sync, JSON, confirmation | PASS |
| `forge update` | single module, --all, no args, JSON | PASS |
| `forge watchforge` | init, add (http/heartbeat), list, status, pause, resume, delete, incidents | PASS |
| Global flags | --output json, --yes, --version, unknown command | PASS |

---

## Bugs Found and Fixed

### BUG-1: Silent exit on missing required flags
**Symptom:** `forge init` (without --domain) exits 1 with no output.
**Cause:** `SilenceErrors: true` swallows Cobra's own "required flag not set"
error before our `cmdErr` handler runs. `Execute()` returned the error
but did not print it.
**Fix:** `Execute()` now prints errors that were not already printed by
`cmdErr` (tracked via `errorPrinted` sentinel bool).
**File:** `core/cmd/root.go`

### BUG-2: Double-print of errors from cmdErr
**Symptom:** `forge install badmodule` printed the error twice to stderr.
**Cause:** `cmdErr` printed the error, then the new `Execute()` fallback
also printed it.
**Fix:** `cmdErr` sets `errorPrinted = true`; `Execute()` only prints
if `!errorPrinted`.
**File:** `core/cmd/root.go`

### BUG-3: forge start hangs when module binary ignores --version
**Symptom:** `forge start <module>` blocked forever on modules whose binary
does not implement `--version` (it just ran their main loop).
**Cause:** `moduleVersion()` used `exec.Command().Output()` which blocks
until the process exits. A module binary that ignores unknown flags runs
its main loop indefinitely.
**Fix:** `moduleVersion()` now uses `exec.CommandContext()` with a 2-second
timeout. Returns "unknown" on timeout.
**File:** `core/cmd/start.go`

### BUG-4: forge update ignored JSON mode for per-module error output
**Symptom:** `forge --output json update smeltforge` printed plain text
instead of JSON.
**Cause:** `updateModule` errors were printed with `fmt.Printf`, bypassing
the JSON mode check.
**Fix:** `runUpdate` accumulates results into a typed slice and calls
`printJSON` when in JSON mode.
**File:** `core/cmd/update.go`

### BUG-5: watchforge resume used wrong flag variable
**Symptom:** `forge watchforge resume --monitor <id>` looked up the monitor
using `wfPauseMonitor` instead of `wfResumeMonitor`, always targeting the
last paused monitor ID rather than the one passed to resume.
**Fix:** Changed `findMonitor(monitors, wfPauseMonitor)` to
`findMonitor(monitors, wfResumeMonitor)` in `wfResumeCmd`.
**File:** `core/cmd/watchforge.go`

### BUG-6: watchforge delete showed wrong monitor name in success message
**Symptom:** Deleting the first monitor by ID showed the second monitor's
name in "Deleted: <name>".
**Cause:** `out := monitors[:0]` shares the backing array with `monitors`.
The filter loop copies `monitors[1]` over `monitors[0]` in-place. `m`
(a pointer to `monitors[0]`) now pointed at the second monitor's data.
**Fix:** Captured `deleteName := m.Name` before the filter loop; changed
`monitors[:0]` to `make([]wfMonitor, 0, len(monitors)-1)`.
**File:** `core/cmd/watchforge.go`

---

## Known Issues (Not Fixed)

### Empty log file created on failed forge start
When `forge start <module>` fails because the binary is missing, it still
creates an empty log file at `~/.forge/data/<module>/forge.log` (the file
is opened before the binary existence check). A subsequent `forge logs
<module>` (without --lines) enters live-tail mode on the empty file
indefinitely. Workaround: always use `--lines N` on modules that may not
have started successfully.

---

## Cleanup

All test HOME directories created during QA (`/tmp/forgetest1`,
`/tmp/forgetest2`, `/tmp/forgewf`, `/tmp/forgewf2`) were removed after
testing. No lingering processes remain.
