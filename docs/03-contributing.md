# Contributing

## Prerequisites

- [mise](https://mise.jdx.dev) — manages Go, Rust, and Just versions
- [Docker](https://docs.docker.com/get-docker/) — required for module integration tests
- Git

## Setup

```bash
# 1. Install mise (one time)
curl https://mise.run | sh

# 2. Clone the repo
git clone https://github.com/<user>/forge.git && cd forge

# 3. Install all pinned tool versions (Go, Rust, Just)
mise install

# 4. Verify
go version       # should match go.work version
rustc --version  # should match .mise.toml rust version
just --version   # should match .mise.toml just version
```

## Building

```bash
# Build everything
just build-all

# Build a single module
just core/build
just fluxforge/build
just smeltforge/build
just watchforge/build
just sparkforge/build
just hearthforge/build
just penforge/build

# Build HearthForge (Go daemon + Rust gateway)
just hearthforge/build
just hearthforge/build-daemon     # Go daemon only
just hearthforge/build-gateway    # Rust gateway only
```

All binaries land in the root `dist/` directory.

## Testing

```bash
# Run all tests
just test-all

# Run tests for a single module
cd <module> && go test ./...

# Run HearthForge tests
cd hearthforge/daemon && go test ./...
cd hearthforge/gateway && cargo test
```

## Code Style

**Go:**
- Format with `gofmt` before committing (`just fmt`)
- Lint with `golangci-lint` (`just lint`)
- Follow standard Go project layout conventions
- All exported functions and types must have doc comments

**Rust (gateway only):**
- Format with `cargo fmt`
- Lint with `cargo clippy`
- Follow Rust API guidelines

## Branch Naming

```
feature/<short-description>
fix/<short-description>
docs/<short-description>
refactor/<short-description>
```

Examples:
- `feature/smeltforge-blue-green`
- `fix/watchforge-ssl-check`
- `docs/penforge-engine-interface`

## Commit Messages

Follow [Conventional Commits](https://www.conventionalcommits.org/):

```
<type>(<scope>): <short description>

[optional body]

[optional footer]
```

Types: `feat`, `fix`, `docs`, `refactor`, `test`, `chore`
Scope: module name (e.g. `smeltforge`, `hearthforge`, `shared`)

Examples:
```
feat(smeltforge): add blue-green deployment strategy
fix(watchforge): correct SSL expiry calculation for wildcard certs
docs(hearthforge): update devctl migration guide
chore(shared): bump golangci-lint version
```

## Pull Requests

- One PR per feature or fix
- All tests must pass (`just test-all`)
- All linting must pass (`just lint`)
- Update relevant docs in `docs/` and `<module>/docs/`
- PRs that touch the `shared/` library need extra care — changes affect all modules

## Adding a New Module

1. Create the directory: `mkdir -p <module>/docs <module>/cmd <module>/internal`
2. Add `go.mod`: `cd <module> && go mod init github.com/<user>/forge/<module>`
3. Add the module to `go.work`
4. Add a `justfile` with at minimum `build`, `test`, `dev` targets
5. Add a `README.md`
6. Add a `main.go` entry point
7. Register the module in Forge core's installer
8. Add a summary doc to `docs/<module>/README.md`
9. Add deep docs to `docs/<module>/` alongside the README

## CI

GitHub Actions runs on every PR:
1. `mise install` — installs pinned tool versions
2. `just test-all` — runs all tests
3. `just lint` — runs golangci-lint
4. Format check — verifies `gofmt` was run

PRs cannot be merged if CI fails.

## Releasing

Releases are triggered by pushing a tag matching `core/v*`. GitHub Actions
builds `forge-linux-amd64` and `forge-linux-arm64`, then publishes a GitHub
Release with both binaries attached.

**Cutting a release:**

```bash
# 1. Update the version string in core/main.go (or wherever version is set)
# 2. Commit and merge to main
# 3. Tag the commit
git tag core/v0.2.0
git push origin core/v0.2.0
```

The release workflow (`.github/workflows/release.yml`) then:
1. Runs `go test ./...` in the `core/` module
2. Cross-compiles for `linux/amd64` and `linux/arm64` with CGO disabled
3. Creates a GitHub Release named "Forge v0.2.0" with both binaries

**Testing install.sh locally:**

```bash
# Syntax check only (no download)
bash -n install.sh

# Full local test against the latest release
curl -fsSL https://raw.githubusercontent.com/Gartner24/forge/main/install.sh | sh
```

**Version tag format:** `core/v<semver>` (e.g. `core/v0.2.0`). Only the core
CLI binary is released this way. Module daemons (`fluxcontroller`, `fluxagent`,
`smeltforged`, etc.) are deployed via `forge install <module>`, not released
as standalone binaries.
