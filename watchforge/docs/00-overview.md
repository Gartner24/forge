# WatchForge

WatchForge monitors the health of your services, containers, cron jobs, and SSL certificates. It maintains a public status page and fires alerts through SparkForge when things go wrong.

## Goals

- Monitor any service — HTTP, TCP, Docker, cron jobs, SSL certs
- Public status page with uptime history and incident log
- Accurate alerting with deduplication (no alert spam)
- Pause monitors automatically during SmeltForge deploys
- Heartbeat monitoring for cron jobs and background tasks

## Architecture

```
WatchForge Daemon
├── monitor registry (registry/monitors.json)
├── check scheduler (goroutine-based, concurrent)
│   ├── HTTP checker
│   ├── TCP checker
│   ├── Docker checker
│   ├── SSL checker
│   └── heartbeat listener
├── incident tracker
├── alert engine (routes to SparkForge)
└── status page generator (static HTML → Caddy)
```

## Concurrency Model

WatchForge uses Go goroutines to run all monitor checks concurrently. Each monitor runs in its own goroutine on its configured interval. This means 200 monitors all run simultaneously without blocking each other — important for accurate timing and avoiding false positives from a slow sequential check loop.

## Incident Lifecycle

```
check fails
  → failure counter increments
  → after N consecutive failures: incident opens, DOWN alert fires
  → check recovers
  → incident closes, RECOVERED alert fires
  → incident written to history
```

Only one DOWN alert fires per incident regardless of how many checks fail. Only one RECOVERED alert fires when the monitor comes back up.

## Status Page

The status page is generated as a static HTML file on every check cycle. It is served by SmeltForge's Caddy instance (or directly if SmeltForge is not installed). Static generation means the status page stays up even if WatchForge itself has a brief issue.

## Deep Documentation

- [Architecture](01-architecture.md) — component details
- [Monitor Types](02-monitor-types.md) — HTTP, TCP, Docker, heartbeat, SSL reference
- [Alert Configuration](03-alerts.md) — priority levels, channels, deduplication
- [Status Page](04-status-page.md) — customization and hosting
- [Operations](05-operations.md) — day-2 management
