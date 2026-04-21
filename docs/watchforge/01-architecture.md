# Architecture

WatchForge is the uptime monitoring daemon. It runs concurrent health checks via goroutines, maintains an append-only incident log, and generates a static HTML public status page.

## Components

### Monitor Scheduler

One goroutine per monitor, each running on its own configured interval. Goroutines are independent — a slow or failing check on one monitor has no effect on others.

Each check goroutine:
1. Executes the check (HTTP request, TCP connect, Docker inspect, etc.)
2. Records the result
3. Applies the failure threshold — an alert fires only after N consecutive failures (configurable, default: 2)
4. Calls `shared/notify` if state changes (DOWN or RECOVERED)
5. Writes to the incident log if an incident opened or closed
6. Triggers status page regeneration

### Incident Tracker

Stateful component that tracks open incidents per monitor. An incident opens on the Nth consecutive failure and closes on the first success. State is persisted to `registry/incidents.json` so incidents survive daemon restarts.

### Status Page Generator

Regenerates the static HTML status page after every check cycle. The page is written atomically (temp file → rename) and served by Caddy. The status page remains accessible even when the WatchForge daemon is down — Caddy serves the last generated file.

## File Layout

```
watchforge/
├── registry/
│   ├── monitors.json       # configured monitors
│   └── incidents.json      # incident state (open/closed)
└── data/
    ├── audit.log
    └── status/
        └── index.html      # generated status page
```

## Integration Points

- **SmeltForge**: calls `watchforge pause/resume` around deploys
- **SparkForge**: DOWN/RECOVERED/WARNING alerts delivered via `shared/notify`
- **HearthForge**: can auto-register container health monitors via `--watch` flag on `add-dev`
