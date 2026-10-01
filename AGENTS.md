# Forge

Self hosted infrastructure suite for solo developers and small teams: one `forge` CLI that manages
mesh networking, deploys, monitoring, notifications, remote dev workspaces and security scanning on
any VPS. Linux only.

## Stack

- **Language / Runtime**: Go 1.26 (workspace of modules via `go.work`), Rust 1.94 (`gateway/`), Python (`mcp-server/`, FastMCP)
- **Framework**: spf13/cobra CLI in `core/cmd/`; module binaries under each module's `cmd/`
- **Key dependencies**: WireGuard (FluxForge), Docker (HearthForge dev containers, MCP server), Nuclei/Nmap/testssl/dnsx (PenForge engines)
- **Package manager**: Go modules; toolchain pinned with mise (`.mise.toml`)

## Build approach

Tracer Bullet (each feature built end to end through CLI, module and docs before the next).

## Commands

```bash
mise install                                   # pinned Go, Rust, Just
go -C core build ./...                         # build one module (repeat per module dir)
go -C core test ./...                          # test one module; release CI runs this for core
cargo build --manifest-path gateway/Cargo.toml # Rust SSH gateway
```

`justfile` is empty: the `just build-all` / `just test-all` targets in `README.md` and
`docs/03-contributing.md` do not exist yet (scope row tracks it).

## Specs

Stored in `docs/specs/`. Format: `docs/specs/NNNN-title.md`. Module reference docs live in `docs/<module>/`.

## Rules

- Every CLI command is non interactive when `--output json` is passed: all required inputs are flags, never TTY prompts (the MCP server calls the CLI via subprocess).
- Commands that run for minutes (PenForge scans) offer `--async` returning an id to poll.
- Most module CLI commands live in `core/cmd/<module>.go`; `core/cmd/penforge.go` delegates to the `penforge` binary with `DisableFlagParsing`. SparkForge commands live only in `sparkforge/cmd/` (no `forge sparkforge` wrapper yet).
- HearthForge is split: `hearthforge/main.go` is the daemon, its CLI is in `core/cmd/hearthforge.go`.
- Go `internal` packages can't be tested from another module: put `_test.go` files inside the package.
- Guard `json.Unmarshal` with `len(data) == 0` when reading state files that may not exist yet.

## Context files

- [docs/00-overview.md](docs/00-overview.md): suite overview; per module docs in `docs/<module>/README.md`

_Drafted by /audit from the repo, worth a quick human pass. Edit freely: once a line stops matching this draft, later runs treat it as curated and will flag rather than overwrite it._
