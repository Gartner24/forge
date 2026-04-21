# Alerting

## Alert States

A monitor transitions through these states:

```
HEALTHY → (N consecutive failures) → DOWN → (first success) → RECOVERED → HEALTHY
```

- `N` is the failure threshold, configurable per monitor (default: 2)
- One DOWN alert fires when the threshold is crossed — not on every failed check
- One RECOVERED alert fires on the first successful check after an incident
- Alerts are delivered via `shared/notify` → SparkForge

## Priority Levels

| Event | Priority |
|---|---|
| Monitor DOWN | `high` |
| SSL cert warning (14 days) | `medium` |
| SSL cert critical (3 days) | `critical` |
| Heartbeat missed | `high` |
| Monitor RECOVERED | `medium` |

## Pausing Monitors

Paused monitors do not execute checks and do not fire alerts. SmeltForge pauses project monitors automatically during deploys.

```bash
forge watchforge pause --monitor <id>
forge watchforge resume --monitor <id>
```

Paused monitors are shown as `PAUSED` on the status page.

## Incident History

Every outage is recorded as an incident:

```bash
forge watchforge incidents                     # all incidents
forge watchforge incidents --monitor <id>      # for a specific monitor
forge watchforge incidents --since 7d
forge watchforge incidents --output json
```

Incident records are never deleted by WatchForge.
