# WatchForge

WatchForge monitors the health of your services, containers, cron jobs, and SSL certificates. It maintains a public status page and fires alerts through SparkForge when things go wrong.

## What It Does

- HTTP/HTTPS endpoint monitoring
- TCP port monitoring
- Docker container health monitoring
- Cron job heartbeat monitoring (did my backup run?)
- SSL certificate expiry monitoring
- Public status page generation
- Incident tracking and history
- Alert integration with SparkForge

## Installation

```bash
forge install watchforge
```

## Quickstart

```bash
# Add an HTTP monitor
forge watchforge add --type http --target https://myapp.com --name "MyApp"

# Check status
forge watchforge list

# View incident history
forge watchforge incidents
```

## Monitor Types

| Type | What it checks |
|---|---|
| `http` | Status code, response time, optional body match |
| `tcp` | Port reachability (useful for databases, Redis) |
| `docker` | Container running state and Docker healthcheck |
| `heartbeat` | Cron job ping URL — alerts if no ping within grace period |
| `ssl` | Certificate expiry date |

## Heartbeat Monitoring

WatchForge generates a unique ping URL per heartbeat monitor. Add it as the last line of your cron job or backup script:

```bash
# Get the ping URL
forge watchforge heartbeat-url --monitor daily-backup

# Add to your script
curl -s https://yourdomain.com/_watchforge/heartbeat/daily-backup/abc123
```

If WatchForge does not receive a ping within the configured grace period, it fires an alert.

## Alert Priority Levels

| Priority | Example | Default channels |
|---|---|---|
| low | Planned maintenance | CLI banner only |
| medium | Service degraded | SparkForge push + webhooks |
| high | Monitor DOWN | SparkForge push + webhooks + email |
| critical | All monitors down, SSL expired | All channels |

Alert deduplication: WatchForge fires one DOWN alert per incident, not one per failed check. No spam.

## Public Status Page

WatchForge generates a static HTML status page served via Caddy. Monitors marked `public: false` never appear on the page.

```
https://status.yourdomain.com

MyApp Status                    🟢 All systems operational
──────────────────────────────────────────────────────────
MyApp Website        🟢 Online   99.98% uptime (30d)
MyApp Container      🟢 Online   100%  uptime (30d)
MyApp SSL            🟢 Valid    Expires in 47 days

Incident History
Mar 13 - 10:00 to 10:08 - MyApp Website (8m 33s)
```

## CLI Reference

```bash
forge watchforge init
forge watchforge add --type <type> --target <target> --name <name>
forge watchforge list
forge watchforge status --monitor <id>
forge watchforge pause --monitor <id>        # maintenance window
forge watchforge resume --monitor <id>
forge watchforge incidents
forge watchforge alerts set --email admin@yourdomain.com
forge watchforge alerts set --webhook https://hooks.slack.com/...
forge watchforge heartbeat-url --monitor <id>
```

## SmeltForge Integration

When SmeltForge is installed:
- Monitors are automatically paused during a SmeltForge deploy — no false down alerts
- SmeltForge can auto-register an HTTP monitor when you add a project: `forge smeltforge add --project myapp --watch`

## Deep Documentation

## Deep Documentation

- [Architecture](01-architecture.md)
- [Monitor Types](02-monitor-types.md)
- [Alerting](03-alerting.md)
- [Status Page](04-status-page.md)
- [CLI Reference](05-cli-reference.md)
