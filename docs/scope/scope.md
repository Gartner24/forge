# Scope: Forge

A self hosted infrastructure suite (mesh, deploys, monitoring, notifications, dev workspaces, security scanning) driven by one `forge` CLI, for solo developers and small teams who want control without SaaS lock in. Parked since 2026-04; this scope records where it stands so it can be picked up again.

**Build approach:** Tracer Bullet (each feature built end to end through CLI, module and docs before the next).
**Workflow:** Beta (after `/develop`: `/check verify`, then `/test`). Secrets, RBAC and network features carry `· GA`.

_These are recommendations to keep your build orderly, not requirements. Skip anything that does not fit. You decide when a feature is `done`._

Every row names its **Source**. Before building a row, read that source; it is the requirement.

## At a glance

| # | Feature | Phase | Status |
|---|---------|-------|--------|
| 1 | Core CLI and secrets store | Existing | existing |
| 2 | FluxForge mesh | Existing | existing |
| 3 | SmeltForge deploys | Existing | existing |
| 4 | WatchForge monitoring | Existing | existing |
| 5 | SparkForge notifications | Existing | existing |
| 6 | PenForge scanning | Existing | existing |
| 7 | HearthForge workspaces and SSH gateway | Existing | existing |
| 8 | Release and install | Existing | existing |
| 9 | MCP server | Slice 1 | in-progress |
| 10 | Build tooling (justfile) | Slice 1 | planned |
| 11 | Tests for SmeltForge and HearthForge | Slice 2 | planned |
| 12 | `forge sparkforge` subcommand | Slice 2 | planned |

## Existing (built before this workflow)

### 1. Core CLI and secrets store · existing
`forge` init, install, uninstall, update, start, stop, status, logs, config, secrets. code in `core/`. Docs `docs/core/`

### 2. FluxForge mesh · existing
WireGuard mesh: init, join, tokens, nodes, admins, ping. code in `fluxforge/`, `core/cmd/fluxforge.go`. Docs `docs/fluxforge/`

### 3. SmeltForge deploys · existing
Deploy, rollback, env vars, webhooks, tokens, polling, deploy keys. code in `smeltforge/`, `core/cmd/smeltforge.go`. Docs `docs/smeltforge/`

### 4. WatchForge monitoring · existing
Monitors, incidents, heartbeat URLs, status page. code in `watchforge/`, `core/cmd/watchforge.go`. Docs `docs/watchforge/`

### 5. SparkForge notifications · existing
Channels, priority routing, send, tokens, alerts. code in `sparkforge/`. Docs `docs/sparkforge/`

### 6. PenForge scanning · existing
Targets, scans (including `--async`), findings, reports, schedules. code in `penforge/`, `core/cmd/penforge.go`. Docs `docs/penforge/`

### 7. HearthForge workspaces and SSH gateway · existing
Dev containers, projects, devs (flag first `add-dev` / `add-project`), Rust SSH gateway. code in `hearthforge/`, `core/cmd/hearthforge.go`, `gateway/`, `templates/`. Docs `docs/hearthforge/`

### 8. Release and install · existing
GitHub release for core (linux amd64 and arm64) and `install.sh`. code in `.github/workflows/release.yml`, `install.sh`

## Slice 1: Make it usable by an agent

### 9. MCP server · in-progress · GA
A Python FastMCP sidecar that wraps the `forge` CLI so an AI agent can run Forge. Only the 8 core tools exist.
**Done when:** every module has its tools; the server ships with `pyproject.toml`, `Dockerfile` and compose as the spec lays out; every tool call is logged as JSONL with secret values redacted; the `forge_install` docstring no longer claims the module starts automatically; the docs warn that `network_mode: host` breaks under `userns-remap`.
- [x] Design it (spec): `/architect MCP server`
- [ ] Build it: `/develop MCP server`
   - [ ] Packaging and server entry (`pyproject.toml`, `Dockerfile`, compose, `/health`)
   - [ ] Module tools (smeltforge, watchforge, sparkforge, fluxforge, hearthforge, penforge)
   - [ ] Audit log in `run_forge()` (TODO-004) and the `forge_install` docstring fix (TODO-003)
   - [ ] `userns-remap` warning and preflight check (TODO-005)
- [ ] Verify it: `/check verify MCP server`
- [ ] Test it: `/test MCP server`
Spec 0002 · code in `mcp-server/`. Source: `docs/specs/0002-mcp-server/`, TODOS.md TODO-003, 004, 005

### 10. Build tooling (justfile)
`README.md` and `docs/03-contributing.md` document `just build-all`, `just test-all` and per module targets, but `justfile` has been empty since it was added.
**Done when:** every documented `just` target exists and runs, or the docs stop promising them.
- [ ] Build it: `/develop build tooling`
Source: `README.md` Quickstart, `docs/03-contributing.md` Building, empty `justfile`

## Slice 2: Fill the gaps

### 11. Tests for SmeltForge and HearthForge
Both modules have no test files; deploys and dev workspace provisioning are the riskiest paths with no coverage.
**Done when:** deploy, rollback and env var handling, and add-dev and add-project validation each have tests that fail when the logic breaks.
- [ ] Test it: `/test SmeltForge and HearthForge`
Source: code scan 2026-10-01 (0 test files in `smeltforge/`, `hearthforge/`)

### 12. `forge sparkforge` subcommand · needs a decision
SparkForge commands exist only in the `sparkforge` binary; every other module is reachable through `forge <module>`.
**Done when:** `forge sparkforge ...` works like the other modules, or the decision to keep it separate is recorded.
- [ ] Design it (spec): `/architect forge sparkforge subcommand`
Source: code scan 2026-10-01 (no `core/cmd/sparkforge.go`)

## Legend

- **Next step** = the first unticked box.
- **needs a decision** = run `/architect` first; otherwise straight to `/develop`.
- **Status** `planned` -> `in-progress` -> `done`, plus `existing` (built before this workflow) and `dropped`.
- **Workflow** Beta: `/check verify` then `/test`. A `· GA` tag adds `/review` (report to `docs/reviews/`) and `/document`.
- **Source** TODOS.md refers to the retired root TODO file, archived at `~/Documents/archive/forge-TODOS-2026-10-01.md`. TODO-001 and TODO-002 are done in code (`core/cmd/hearthforge.go`, `penforge/cmd/scan.go`).
