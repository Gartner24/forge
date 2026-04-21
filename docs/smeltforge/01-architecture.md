# Architecture

SmeltForge is the deployment platform. It manages Docker containers and Compose stacks, handles routing via a built-in Caddy reverse proxy, and provides multiple deploy trigger mechanisms.

## Components

### SmeltForge Daemon (Go)

The core process. Handles all deployment operations:
- Project registry management
- Source pulling (git clone/pull, docker pull, local directory)
- Container lifecycle (stop/start or blue-green)
- Environment variable injection from `forge secrets`
- Deploy audit logging
- Webhook and CI token request handling
- Integration hooks to WatchForge, SparkForge, and PenForge

### Caddy

A Caddy instance runs alongside the daemon, managed by SmeltForge via Caddy's JSON Admin API. SmeltForge never writes a Caddyfile directly — all routing changes go through the API. Caddy handles:
- Automatic TLS via Let's Encrypt (HTTPS for all configured domains)
- Reverse proxy routing from domain → container port
- Zero-downtime route switching for blue-green deploys
- Dev preview domain routing for HearthForge (when both are installed)

## Data Flow

```
deploy trigger (CLI / webhook / polling / CI token)
        ↓
SmeltForge daemon validates and queues
        ↓
WatchForge: pause monitors for this project
        ↓
Source: git pull / docker pull
        ↓
Container: stop-start or blue-green
        ↓
Secrets: inject env vars at container start
        ↓
Caddy: update routing (zero-downtime)
        ↓
WatchForge: resume monitors
        ↓
SparkForge: send result notification
        ↓
PenForge: trigger post-deploy scan (if configured)
        ↓
Audit: write deploy record
```

## Deploy Queue

If a deploy is already in progress when a new trigger arrives (webhook or polling), the new trigger is queued and processed when the current deploy completes. Queue depth is configurable (default: 5 pending deploys per project).

## File Layout

```
smeltforge/
├── registry/
│   └── projects.json       # registered projects
├── data/
│   ├── deploys.log         # append-only deploy audit log
│   └── workspaces/         # git working copies per project
└── caddy/
    └── config/             # Caddy JSON config fragments (managed by daemon)
```
