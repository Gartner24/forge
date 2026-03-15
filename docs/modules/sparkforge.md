# SparkForge

SparkForge is the notification layer for the Forge suite. It routes alerts from all modules to configured channels based on priority level. It wraps Gotify for mobile push notifications and also supports email and webhooks.

> **SparkForge is never a hard dependency.** If it is not installed, each module falls back to its own basic alerting. SparkForge is an enhancement, not a requirement.

## What It Does

- Routes alerts from all Forge modules to the right channels
- Manages Gotify (self-hosted push notifications) automatically
- Supports email (SMTP), webhooks (Slack, Discord, custom HTTP), and mobile push
- Exposes a public API so your own scripts and apps can send notifications
- Shows active alerts in the Forge CLI banner
- Deduplicates alerts — no spam for the same incident

## Installation

```bash
forge install sparkforge
# → pulls and starts Gotify as a Docker container
# → prints Gotify URL, admin credentials, and mobile app setup instructions
```

## Channels

| Channel | Backend | Default minimum priority |
|---|---|---|
| Push (mobile) | Gotify | low |
| Webhook | Slack, Discord, custom HTTP | medium |
| Email | SMTP | high |
| In-CLI banner | forge CLI | high |

Each channel has a configurable `priority_min`. Messages below that priority are silently dropped for that channel. This prevents low-priority deploy notifications from waking you up at 3am while still sending critical alerts everywhere.

## Priority Levels

| Priority | Example |
|---|---|
| low | Deploy started, environment provisioned |
| medium | Deploy succeeded/failed, service degraded |
| high | Monitor DOWN, repeated auth failures |
| critical | All monitors down, SSL expired, gateway down |

## Configuration

```bash
# Add a Slack webhook channel
forge sparkforge channel add \
  --type webhook \
  --url https://hooks.slack.com/... \
  --name slack-deploys \
  --priority-min medium

# Add an email channel
forge sparkforge channel add \
  --type email \
  --to admin@yourdomain.com \
  --priority-min high

# List configured channels
forge sparkforge channel list

# Disable a channel temporarily
forge sparkforge channel disable slack-deploys
```

## Public API

Any script or app on the server can send through SparkForge:

```bash
curl -X POST https://localhost:7778/notify \
  -H "Authorization: Bearer <token>" \
  -d '{
    "title": "Backup completed",
    "message": "Daily backup finished in 4m 32s",
    "priority": "low",
    "source": "backup-script"
  }'
```

Generate an API token:
```bash
forge sparkforge token create
```

## CLI Reference

```bash
forge sparkforge init
forge sparkforge channel add --type <type> --name <n>
forge sparkforge channel list
forge sparkforge channel enable <id>
forge sparkforge channel disable <id>
forge sparkforge send --title <t> --message <m> --priority <p>
forge sparkforge logs
forge sparkforge token create
forge sparkforge token revoke <token>
```

## Deep Documentation

See [`sparkforge/docs/`](../../sparkforge/docs/) for:
- Channel configuration reference
- Priority routing rules
- Gotify setup and mobile app guide
- Public API reference
- Alert deduplication internals
