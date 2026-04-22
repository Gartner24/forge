# Forge

Forge is a self-hosted infrastructure suite for solo developers and small teams who want full control over their infrastructure without vendor lock-in. It replaces a collection of paid SaaS tools with self-hosted, modular, integrated alternatives that run on any VPS or server.

## The Suite

| Module | Name | Role |
|---|---|---|
| Core | **Forge** | CLI binary, secrets store, module manager |
| Mesh network | **FluxForge** | WireGuard-based private mesh networking |
| Deployment | **SmeltForge** | Deploy apps across your servers |
| Monitoring | **WatchForge** | Uptime monitoring and public status page |
| Notifications | **SparkForge** | Alerts via Gotify, email, and webhooks |
| Dev workspaces | **HearthForge** | Remote developer environments via SSH |
| Security | **PenForge** | Automated security scanning |

## MCP Server

Forge ships an MCP server sidecar (`mcp-server/`) that lets AI agents (Claude, etc.)
control your infrastructure via natural language. It wraps the `forge` CLI binary via
subprocess and exposes all 7 modules as MCP tools over Streamable HTTPS.

```bash
# Start the MCP server (requires forge CLI installed on the host)
cd mcp-server
docker compose up -d
```

Connect Claude Code: add `https://localhost:8008/mcp` as an MCP server endpoint.
See `mcp-server/` for full setup instructions.

## Quickstart

```bash
# 1. Install mise (manages Go, Rust, Just versions)
curl https://mise.run | sh

# 2. Clone the repo
git clone https://github.com/<user>/forge.git && cd forge

# 3. Install all pinned tool versions
mise install

# 4. Build everything
just build-all

# 5. Run all tests
just test-all
```

## Installing Forge

```bash
# One-liner install (Linux only, amd64 and arm64)
curl -fsSL https://raw.githubusercontent.com/Gartner24/forge/main/install.sh | sh
```

Or build from source:

```bash
# Install Forge core (CLI + secrets store — always the first step)
just core/install

# Then install whichever modules you need
forge install smeltforge
forge install watchforge
forge install sparkforge
forge install hearthforge
forge install penforge

# FluxForge only if you need multi-VPS mesh networking
forge install fluxforge

# Check what is installed and running
forge status
```

## Repository Layout

```
forge/
├── README.md
├── .mise.toml          # pins Go, Rust, Just versions
├── go.work             # Go workspace (ties all modules together locally)
├── justfile            # root task runner
├── docs/               # all documentation lives here
│   ├── 00-overview.md ... 05-security.md  # suite-level docs
│   ├── shared/         # shared library docs (module/, notify/, audit/, secrets/, registry/, config/)
│   ├── core/           # Forge Core docs
│   ├── fluxforge/      # FluxForge docs
│   ├── smeltforge/     # SmeltForge docs
│   ├── watchforge/     # WatchForge docs
│   ├── sparkforge/     # SparkForge docs
│   ├── hearthforge/    # HearthForge docs (14 files)
│   └── penforge/       # PenForge docs
├── shared/             # shared Go libraries
├── core/               # Forge CLI + secrets store
├── fluxforge/          # mesh networking
├── smeltforge/         # deployment platform
├── watchforge/         # uptime monitoring
├── sparkforge/         # notifications
├── hearthforge/        # remote dev workspaces
│   ├── daemon/         # Go provisioning daemon
│   └── gateway/        # Rust SSH gateway
├── penforge/           # security scanning
└── mcp-server/         # Python/FastMCP sidecar -- AI agent interface to the full suite
```

## Documentation

**Suite-level docs** (start here):
- [Overview](docs/00-overview.md)
- [Architecture](docs/01-architecture.md)
- [Project Structure](docs/02-project-structure.md)
- [Contributing](docs/03-contributing.md)
- [Releasing](docs/04-releasing.md)
- [Security Policy](docs/05-security.md)

**Module docs:**
- [Shared Library](docs/shared/README.md)
- [Forge Core](docs/core/README.md)
- [FluxForge](docs/fluxforge/README.md)
- [SmeltForge](docs/smeltforge/README.md)
- [WatchForge](docs/watchforge/README.md)
- [SparkForge](docs/sparkforge/README.md)
- [HearthForge](docs/hearthforge/README.md)
- [PenForge](docs/penforge/README.md)

## License

MIT
