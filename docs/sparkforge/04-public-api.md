# Public Notification API

SparkForge exposes a local HTTP API at `localhost:7778/notify`. External scripts, cron jobs, and applications running on the server can send notifications through this endpoint — they do not need to call `shared/notify` directly.

## Authentication

All requests require a Bearer token in the Authorization header. Unauthenticated requests return HTTP 401.

```bash
# Create a token
forge sparkforge token create --name "backup-script"

# List tokens
forge sparkforge token list

# Revoke a token
forge sparkforge token revoke <token-id>
```

Tokens are stored in `forge secrets` under `sparkforge.api_tokens.*`. They are never shown after creation — store them immediately.

## Sending a Notification

```
POST http://localhost:7778/notify
Authorization: Bearer <token>
Content-Type: application/json

{
  "title": "Backup completed",
  "body": "Full backup of /opt/data completed in 4m 32s. Size: 2.1GB",
  "priority": "low",
  "source": "backup-script",
  "link": "https://example.com/backup-logs"
}
```

Fields:
| Field | Required | Description |
|---|---|---|
| `title` | Yes | Short headline (≤80 chars) |
| `body` | No | Full message text |
| `priority` | Yes | `low`, `medium`, `high`, or `critical` |
| `source` | Yes | Identifies who is sending — used for deduplication |
| `link` | No | Optional URL to attach |

Response: `200 OK` with `{"status": "delivered", "channels": ["gotify-mobile"]}` on success.

## Usage in Scripts

```bash
#!/bin/bash
# Backup script

perform_backup || exit 1

curl -s -X POST http://localhost:7778/notify \
  -H "Authorization: Bearer $FORGE_API_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "title": "Backup succeeded",
    "priority": "low",
    "source": "backup-cron"
  }'
```

Store the token in `forge secrets` and read it in your script:
```bash
FORGE_API_TOKEN=$(forge secrets get sparkforge.api_tokens.backup-script)
```

## Combining with WatchForge Heartbeat

A robust pattern for cron job monitoring:

```bash
#!/bin/bash
# Run the job
perform_backup

if [ $? -eq 0 ]; then
  # Notify SparkForge (low priority — informational)
  curl -s -X POST http://localhost:7778/notify \
    -H "Authorization: Bearer $TOKEN" \
    -d '{"title":"Backup done","priority":"low","source":"backup-cron"}'
  # Ping WatchForge heartbeat (proves the job ran on time)
  curl -s https://status.example.com/_ping/abc123
else
  # Send a high priority alert on failure
  curl -s -X POST http://localhost:7778/notify \
    -H "Authorization: Bearer $TOKEN" \
    -d '{"title":"Backup FAILED","priority":"high","source":"backup-cron"}'
  # Do NOT ping heartbeat — WatchForge will alert separately when the ping is missed
fi
```
