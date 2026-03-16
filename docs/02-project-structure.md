# Project Structure

Forge is a public GitHub monorepo. Each module lives in its own top-level directory with its own Go module (`go.mod`), justfile, README, and docs. A root-level `go.work` file ties all Go modules together locally.

## Root Layout

```
forge/
├── README.md
├── .mise.toml              # pins Go, Rust, Just versions
├── go.work                 # Go workspace
├── go.work.sum
├── justfile                # root task runner
├── .gitignore
├── .github/
│   └── workflows/
│       ├── ci.yml          # runs on every PR
│       ├── release.yml     # runs on version tag push
│       └── security.yml    # weekly PenForge scan
├── docs/                   # all documentation lives here
│   ├── 00-overview.md ... 05-security.md  # suite-level docs
│   ├── shared/             # shared library docs (module, notify, audit, secrets, registry, config)
│   ├── core/               # Forge Core docs
│   ├── fluxforge/          # FluxForge docs
│   ├── smeltforge/         # SmeltForge docs
│   ├── watchforge/         # WatchForge docs
│   ├── sparkforge/         # SparkForge docs
│   ├── hearthforge/        # HearthForge docs (14 files)
│   └── penforge/           # PenForge docs
├── shared/                 # shared Go libraries
├── core/                   # Forge CLI + secrets store
├── fluxforge/              # mesh networking
├── smeltforge/             # deployment platform
├── watchforge/             # uptime monitoring
├── sparkforge/             # notifications
├── hearthforge/            # remote dev workspaces
│   ├── daemon/             # Go provisioning daemon
│   └── gateway/            # Rust SSH gateway
└── penforge/               # security scanning
```

## Tool Versioning — .mise.toml

Mise ensures every contributor and every CI run uses identical versions of Go, Rust, and Just.

```toml
[tools]
go   = "1.23.4"
rust = "1.75.0"
just = "1.27.0"

[settings]
experimental = true
```

After cloning: `mise install`

Modules do **not** each need their own `.mise.toml`. The root file applies to all subdirectories automatically.

## Go Workspace — go.work

The `go.work` file lets all Go modules in the repo import `shared/` and each other locally without publishing to pkg.go.dev.

```
go 1.23

use (
    ./core
    ./shared
    ./fluxforge
    ./smeltforge
    ./watchforge
    ./sparkforge
    ./hearthforge/daemon
    ./penforge
)
```

## Root justfile

The root justfile handles suite-wide operations. Module justfiles handle module-specific operations.

```just
default:
    @just --list

build-all:
    just core/build
    just fluxforge/build
    just smeltforge/build
    just watchforge/build
    just sparkforge/build
    just hearthforge/build
    just penforge/build

test-all:
    go test ./...

lint:
    golangci-lint run ./...

fmt:
    gofmt -w .

release module version:
    git tag {{module}}/v{{version}}
    git push origin {{module}}/v{{version}}

clean:
    rm -rf dist/
    cd hearthforge/gateway && cargo clean
```

## Module Structure

Every module follows the same pattern:

```
<module>/
├── go.mod          # own Go module: github.com/<user>/forge/<module>
├── justfile        # at minimum: build, test, dev
├── README.md       # what it does, how to build, how to run
├── main.go         # entry point
├── cmd/            # cobra CLI subcommands (one file per command)
├── internal/       # private business logic (not importable externally)
└── registry/       # JSON config files (not committed with real data)
```

Documentation for each module lives in `docs/<module>/` at the repo root, not inside the module directory itself. This keeps all documentation in one place and the module directories focused purely on code.

### Directory Conventions

- `cmd/` — cobra CLI command definitions. One file per command. Public.
- `internal/` — business logic. Not importable by other modules. Private.
- `registry/` — JSON data files (projects, monitors, targets). Example entries only in git.
- `tests/` — integration tests that require external dependencies (Docker, network).

### HearthForge is Different

HearthForge has two build systems (Go and Rust) so its layout differs slightly:

```
hearthforge/
├── justfile            # builds both daemon and gateway
├── README.md
├── daemon/             # Go provisioning daemon
│   ├── go.mod
│   ├── main.go
│   ├── cmd/
│   └── internal/
├── gateway/            # Rust SSH gateway
│   ├── Cargo.toml
│   ├── Cargo.lock
│   ├── Dockerfile
│   └── src/
├── templates/          # container and proxy config templates
└── registry/           # devs.json, projects.json
```

HearthForge documentation lives in `docs/hearthforge/` alongside all other module docs.

## shared/ Library

The shared library contains packages imported by every module. No business logic — only reusable infrastructure code.

```
shared/
├── go.mod
├── audit/      # append-only audit log writer
├── secrets/    # secrets store client
├── registry/   # registry file parsers
├── notify/     # SparkForge HTTP client
├── config/     # global forge config reader
└── module/     # Module interface all modules must implement
```

**Rule:** if more than one module needs the same code, it goes in `shared/`. If only one module needs it, it stays in that module's `internal/`.

Full documentation for each package lives in [`docs/shared/`](../docs/shared/README.md). If you are building a new module, read `docs/shared/` before writing any code — understanding the Module interface, the notify client, and the audit writer up front will save significant rework.

## Naming Conventions

- Go module path: `github.com/<user>/forge/<module-dir>`
- Binary output: `dist/<binary-name>` (all binaries land in root `dist/`, gitignored)
- Go files: `snake_case.go`
- Go test files: `snake_case_test.go`
- Config files: `kebab-case.json` / `kebab-case.toml`
- Template files: `descriptive-name.tmpl`

## Every Module Must Have

| File | Purpose |
|---|---|
| `go.mod` | Own Go module definition |
| `justfile` | At minimum: `build`, `test`, `dev` targets |
| `README.md` | What it does, how to build it, how to run it |
| `main.go` | Entry point (for binary modules) |

## Contributor Quickstart

```bash
# 1. Install mise (one time)
curl https://mise.run | sh

# 2. Clone
git clone https://github.com/<user>/forge.git && cd forge

# 3. Install pinned tool versions
mise install

# 4. Build everything
just build-all

# 5. Run all tests
just test-all
```

See [Contributing](03-contributing.md) for the full contributor guide.
