# Priority Routing and Deduplication

## Priority Levels

| Level | Value | Typical use |
|---|---|---|
| `low` | 1 | Informational events — deploy started, scan scheduled |
| `medium` | 2 | Completed operations — deploy succeeded, monitor recovered |
| `high` | 3 | Requires prompt attention — deploy failed, monitor down, high CVE found |
| `critical` | 4 | Requires immediate action — SSL cert expiring in 3 days, gateway down |

## Channel Routing

Each channel has a `priority_min` setting. A message is delivered to a channel only if `message.priority >= channel.priority_min`.

Example setup — two channels with different thresholds:

| Channel | Type | priority_min |
|---|---|---|
| gotify-mobile | Gotify | low (receives everything) |
| slack-critical | Webhook | high (only HIGH and CRITICAL) |

A `medium` priority message from SmeltForge ("deploy succeeded") goes to Gotify but not to Slack. A `critical` message ("SSL cert expiring") goes to both.

Update a channel's threshold:
```bash
forge sparkforge channel update <id> --priority-min high
```

## Alert Deduplication

SparkForge tracks active alerts by `(source, event_type)` key. If WatchForge fires a DOWN alert for `monitor:api-health`, SparkForge delivers it to channels and records it as active. If WatchForge fires another DOWN alert for the same monitor before it recovers, SparkForge drops it silently — the admin already knows.

The alert clears when:
- A RECOVERED notification is sent for the same `(source, event_type)`
- The alert is acknowledged in SparkForge (manual clear)

```bash
forge sparkforge alerts list                  # active alerts
forge sparkforge alerts acknowledge <id>      # manually clear an alert
```

## In-CLI Alert Banner

When any active alert has priority `high` or `critical`, Forge Core prepends an alert banner to every CLI command output:

```
⚠  ACTIVE ALERTS
   [HIGH]     watchforge  monitor:api-health     DOWN — 23 minutes ago
   [CRITICAL] watchforge  monitor:hemis-ssl       SSL expiry in 2 days

forge smeltforge status
...
```

The banner clears automatically when all high/critical alerts resolve. Suppress it for a single command:

```bash
forge status --no-banner
```
