# Architecture

SparkForge is the notification orchestration layer. It routes alerts from all Forge modules to configured delivery channels based on priority level. It wraps Gotify for mobile push notifications and adds email and webhook delivery on top.

## Components

### SparkForge Daemon (Go)

The core process. Responsibilities:
- Receives notification requests from modules via `shared/notify`
- Receives external notification requests via the public HTTP API
- Applies priority-based routing to determine which channels receive each message
- Deduplicates alerts — identical active alerts are not re-delivered
- Manages channel configuration
- Manages the in-CLI alert banner state
- Writes delivery audit log

### Gotify

A Gotify instance runs as a Docker container alongside the daemon, fully managed by SparkForge. Admins never interact with Gotify directly. SparkForge creates Gotify apps, manages tokens, and delivers messages to it via Gotify's REST API. Gotify handles push delivery to mobile devices via the Gotify Android/iOS app.

## Message Flow

```
module calls shared/notify.Send(msg)
        ↓
SparkForge daemon receives via internal HTTP
        ↓
Deduplication check — is an identical active alert already open?
  [yes] → drop silently
  [no]  → continue
        ↓
Priority routing — which channels have priority_min ≤ msg.priority?
        ↓
Deliver to each matching channel concurrently
  ├── Gotify → mobile push
  ├── SMTP → email
  └── Webhook → Slack / Discord / custom
        ↓
Write delivery record to audit log
        ↓
Update in-CLI banner state if priority ≥ HIGH
```

A failure in one channel does not block delivery to others.

## File Layout

```
sparkforge/
├── registry/
│   └── channels.json       # configured notification channels
└── data/
    ├── delivery.log        # append-only delivery audit log
    └── alerts.json         # active alert state (for deduplication + banner)
```

## Integration Points

- **All modules**: call `shared/notify.Send()` — SparkForge is the only consumer
- **Forge Core**: reads `alerts.json` to render the in-CLI alert banner
- **External scripts**: can POST to `localhost:7778/notify` with a valid API token
