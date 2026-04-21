# Notification Channels

A channel is a delivery destination. SparkForge supports three types: Gotify (mobile push), email (SMTP), and webhook (Slack, Discord, or any HTTP endpoint).

## Gotify (Mobile Push)

Gotify is automatically installed and configured during `forge sparkforge init`. Admins install the Gotify app on their phone, connect it to the Gotify server, and receive push notifications for all alerts routed to this channel.

```bash
# Gotify is configured automatically — no manual setup needed
forge install sparkforge
# → installs SparkForge + Gotify, prints the Gotify URL and app token
```

To get the mobile connection URL:
```bash
forge sparkforge gotify show
```

## Email (SMTP)

```bash
forge sparkforge channel add \
  --type email \
  --name "alerts-email" \
  --smtp-host smtp.example.com \
  --smtp-port 587 \
  --smtp-user alerts@example.com \
  --smtp-password "..." \
  --to admin@example.com \
  --priority-min medium
```

SparkForge validates SMTP connectivity when the channel is added. If the connection fails, the channel is saved but flagged as unreachable — run `forge sparkforge channel test <id>` to retry.

## Webhook

```bash
forge sparkforge channel add \
  --type webhook \
  --name "slack-alerts" \
  --url https://hooks.slack.com/services/... \
  --priority-min high
```

The webhook receives a POST with a JSON body:
```json
{
  "title": "DOWN: API health",
  "body": "Failed 2 consecutive checks. Last error: connection refused",
  "priority": "high",
  "source": "watchforge",
  "timestamp": "2026-03-14T10:23:01Z"
}
```

For Slack and Discord, SparkForge formats the payload to match their expected webhook format automatically when the URL pattern is recognised.

## Managing Channels

```bash
forge sparkforge channel list
forge sparkforge channel show <id>
forge sparkforge channel delete <id>
forge sparkforge channel enable <id>
forge sparkforge channel disable <id>       # suspend without deleting
forge sparkforge channel test <id>          # send a test notification
forge sparkforge channel update <id> --priority-min critical
```

## Sending a Test Notification

```bash
forge sparkforge send --title "Test" --priority medium
forge sparkforge send --title "Test" --channel <id>   # specific channel only
```
