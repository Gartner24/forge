# SmeltForge

SmeltForge is the deployment platform for the Forge suite. It deploys Docker Compose stacks and single containers, manages a Caddy reverse proxy for automatic TLS, and integrates with WatchForge and PenForge for automated monitoring and security scanning on every deploy.

## Goals

- Deploy any Docker-based application with one command
- Automatic TLS via Caddy + Let's Encrypt
- Zero-downtime deploys (blue-green strategy)
- Secure environment variable management via `forge secrets`
- Full deploy audit trail
- Trigger deploys from git push, webhooks, polling, or CI

## Why Caddy Instead of Nginx

SmeltForge uses Caddy as its built-in reverse proxy. Caddy:
- Handles TLS automatically — no certbot, no cron jobs, no cert renewal scripts
- Reloads via API with zero downtime — SmeltForge can update routing instantly without touching files
- Ships as a single lightweight binary

If HearthForge is installed, it shares SmeltForge's Caddy instance for dev preview domains — one proxy on the server instead of two.

## Architecture

```
SmeltForge Daemon
├── project registry (registry/projects.json)
├── deploy engine
│   ├── stop-start strategy
│   └── blue-green strategy
├── Caddy API client (proxy management)
├── webhook listener
├── git poller
└── deploy audit log

Caddy (Docker container, managed by SmeltForge)
├── automatic TLS
└── domain → container routing
```

## Deploy Lifecycle

```
trigger (manual / webhook / polling / CI)
  → validate project exists
  → pull source (git clone / docker pull / local)
  → run deploy strategy
      stop-start: stop old → start new
      blue-green: start new → health check → switch → stop old
  → update Caddy routing
  → inject env vars from forge secrets
  → write audit log entry
  → notify SparkForge (if installed)
  → trigger WatchForge monitor resume (if installed)
  → trigger PenForge post-deploy scan (if configured)
```

## Deep Documentation

- [Architecture](01-architecture.md) — component details
- [Project Registry](02-project-registry.md) — projects.json schema reference
- [Deploy Strategies](03-deploy-strategies.md) — stop-start and blue-green internals
- [Webhook Security](04-webhooks.md) — securing webhook endpoints
- [Environment Variables](05-env-vars.md) — secrets injection at deploy time
- [Audit Trail](06-audit-trail.md) — deploy log format
- [Operations](07-operations.md) — day-2 management
