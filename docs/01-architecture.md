# Architecture

This document describes the full Forge system design, components, language decisions, and how everything fits together.

## System Summary

Forge is a monorepo of independent modules. Each module is a self-contained Go binary (except the HearthForge SSH gateway which is Rust) that can be installed and run independently. Modules communicate through shared filesystem state, the `forge secrets` store, and SparkForge's notification API — never by importing each other directly at runtime.

## Language Decisions

| Component | Language | Reason |
|---|---|---|
| Forge core CLI | Go | cobra CLI framework, fast iteration |
| FluxForge controller + agent | Go | WireGuard Go ecosystem (same as Tailscale) |
| SmeltForge | Go | Docker SDK is Go-first |
| WatchForge | Go | Goroutines ideal for concurrent checks |
| SparkForge | Go | Simple HTTP orchestration |
| HearthForge daemon | Go | Docker API usage |
| HearthForge SSH gateway | **Rust** | Security-critical, no GC pauses, russh library |
| PenForge | Go | Docker orchestration, JSON parsing |
| ForgeScanner (future) | **Rust** | Low-level networking, performance-critical |

**Why Rust for the SSH gateway specifically:**
The gateway sits on a public SSH port and handles many concurrent connections 24/7. Rust eliminates memory corruption vulnerabilities at compile time, has no GC pauses (consistent connection latency), and the `russh` library gives lower-level SSH protocol control than Go equivalents. Every other module uses Go because the tradeoffs flip — orchestration, HTTP, and Docker API work benefits from Go's faster iteration and richer ecosystem.

## Components

### Forge Core
The CLI binary and encrypted secrets store. The only thing installed by default. Manages module installation and the `forge secrets` keystore. Provides the in-CLI alert banner that surfaces active WatchForge/PenForge findings before any command output.

### Shared Library (`shared/`)
Go packages imported by all modules:
- `audit/` — append-only audit log writer
- `secrets/` — namespaced secrets store client
- `registry/` — project and dev registry file parsers
- `notify/` — SparkForge HTTP client (no-op if SparkForge not installed)
- `config/` — global forge config reader
- `module/` — Module interface every installed module must implement

### FluxForge
WireGuard mesh networking. Two binaries:
- **FluxController** — coordination server. Runs on one node. Manages peer registry, join tokens, admin API, and acts as a NAT relay of last resort.
- **FluxAgent** — runs on every node. Registers with controller, syncs peer list, configures WireGuard. Sends periodic heartbeats.

Every node gets a stable private IP in the `10.forge.x.x` range. Other modules reference nodes by mesh IP.

### SmeltForge
Deployment platform. Manages Docker Compose stacks and single containers. Uses Caddy as a built-in reverse proxy for automatic TLS. Supports deploy-from-git, deploy-from-registry, and deploy-from-local. Deploy triggers: manual CLI, GitHub/GitLab webhook, git polling, CI token.

### WatchForge
Uptime monitoring daemon. Runs concurrent health checks via goroutines. Monitor types: HTTP/HTTPS, TCP, Docker container health, cron heartbeats, SSL certificate expiry. Maintains append-only incident log. Generates a static HTML public status page served via Caddy.

### SparkForge
Notification orchestration layer over Gotify. Routes messages from all modules to configured channels by priority level (low / medium / high / critical). Channels: Gotify push (mobile), email (SMTP), webhook (Slack, Discord, custom HTTP), in-CLI banner. Exposes a public API so non-Forge scripts can send notifications.

### HearthForge
Remote developer workspaces. Two components:
- **Daemon (Go)** — provisions Docker containers per (developer, project). Manages SSH keys, vhost config, deploy keys, and container lifecycle.
- **Gateway (Rust)** — SSH jump host. Authenticates developers via public key, routes connections to their dev container's sshd. Never provides a host shell.

One container per (developer, project). Containers run as non-root, attach only to `dev-web` network, never have Docker socket access.

### PenForge
Security scanning orchestrator. Runs scan engines (Nuclei, Nmap, testssl.sh, dnsx, Trivy) as isolated Docker containers. Enforces scope — only scans registered targets. Manages finding lifecycle (new → acknowledged → fixed → verified). Generates structured scan reports. Integrates with SmeltForge for post-deploy scans.

## Cross-Module Integration

| Module A | Module B | What the integration adds |
|---|---|---|
| SmeltForge | WatchForge | Auto-pause monitors during deploy, auto-register monitors on project add |
| SmeltForge | PenForge | Post-deploy security scan hook |
| SmeltForge | SparkForge | Deploy success/failure notifications |
| SmeltForge | HearthForge | Shared Caddy proxy for dev preview domains |
| WatchForge | SparkForge | Down/recovered/warning alerts |
| HearthForge | SparkForge | SSH auth failure alerts, gateway down alerts |
| HearthForge | WatchForge | Auto-register container health monitors |
| PenForge | SparkForge | New critical finding alerts |
| All modules | FluxForge | Multi-node deployment, mesh IP addressing, cross-node secrets sync |
| All modules | forge secrets | Namespaced encrypted secret storage |

All integrations are opt-in and graceful — if the target module is not installed, the integration silently does nothing.

## Audit Logging

Every module writes structured audit events through `shared/audit`. All logs are:
- Append-only (enforced by file permissions)
- Never deleted or truncated by Forge
- Stored per-module at paths configured in `~/.forge/config.toml`

Minimum fields per event: timestamp, module, event type, actor identity, target, result.

## Secrets Management

`forge secrets` is a core built-in command backed by an age-encrypted file on disk. Zero storage overhead — just one encrypted file.

Secrets are namespaced per module:
```
smeltforge.myapp.DATABASE_URL
hearthforge.deploykeys.myproject
fluxforge.controller.token
```

With FluxForge installed, secrets gain `--sync` flag to replicate across mesh nodes over the private `10.forge.x.x` network.

## Security Boundaries

- No `--privileged` containers anywhere in the suite
- Docker socket is never mounted into any module or dev container
- Dev containers run as non-root, attach only to `dev-web` network
- PenForge scan engines run in isolated containers with no access to internal networks
- FluxForge mesh uses WireGuard — all traffic encrypted in transit
- HearthForge SSH gateway provides no host shell — jump-host only

## Build Order

Modules should be built in this order — each phase delivers standalone value:

1. Forge Core — CLI, secrets, module manager
2. FluxForge — mesh foundation
3. SmeltForge — most immediately useful, highest daily-use value
4. WatchForge — pairs naturally with SmeltForge deploys
5. SparkForge — short build, unlocks alerts for all previous modules
6. HearthForge — migration of existing work
7. PenForge — depends on SmeltForge post-deploy hooks
8. Autoscaling (server scaling, stateless only) — after multi-server deployment is stable
9. ForgeScanner — custom Rust scan engine replacing Nuclei

## Future Architecture Notes

- **Full HA FluxForge** — Raft consensus across 3 controller nodes
- **Database scaling** — read replicas, then multi-DB strategies (after multi-server is stable)
- **Web dashboard** — after CLI is solid across all modules
- **DNS inside mesh** — stable hostnames (`vps1.forge`) instead of IPs
- **ForgeScanner** — custom Rust engine implementing PenForge's Engine interface
