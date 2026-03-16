# SparkForge

SparkForge is the notification layer for the Forge suite. It routes alerts from all modules to configured channels based on priority level. It wraps Gotify for mobile push notifications and also supports email and webhooks.

## Goals

- Single notification entry point for the entire Forge suite
- Route alerts to the right channels at the right priority
- No alert spam — deduplication built in
- Mobile push notifications via Gotify (self-hosted)
- Public API so non-Forge scripts can also send notifications

## Architecture

```
SparkForge Daemon
├── channel registry (registry/channels.json)
├── message router
│   └── priority filter per channel
├── channels
│   ├── Gotify (push)
│   ├── SMTP (email)
│   └── webhook (HTTP)
├── alert deduplicator
├── public HTTP API
└── delivery log

Gotify (Docker container, managed by SparkForge)
└── mobile push notifications
```

## Message Flow

```
module fires alert (e.g. WatchForge: myapp is DOWN)
  → SparkForge receives message with priority=high
  → deduplicator checks: is this a duplicate of an open incident?
  → if not duplicate: route to all channels where priority_min <= high
      → Gotify: send push ✓
      → Slack webhook: send ✓
      → email: send ✓
  → write to delivery log
```

## Gotify

Gotify is an open-source self-hosted push notification server. SparkForge installs and manages it automatically as a Docker container. Developers install the Gotify mobile app and subscribe to their server to receive push notifications on their phone.

SparkForge manages Gotify behind the scenes — admins never need to interact with Gotify directly.

## Deep Documentation

- [Architecture](01-architecture.md) — component details
- [Channel Configuration](02-channels.md) — all channel types and options
- [Priority Routing](03-priority-routing.md) — how priorities map to channels
- [Public API](04-api.md) — sending notifications from external scripts
- [Gotify Setup](05-gotify.md) — mobile app setup guide
- [Operations](06-operations.md) — day-2 management
