# Monitor Types

## HTTP / HTTPS

Checks a URL for a valid response.

```bash
forge watchforge add \
  --type http \
  --name "API health" \
  --target https://api.example.com/health \
  --interval 60 \
  --public
```

Options:
- `--expected-status <code>` — expected HTTP status code (default: 200)
- `--contains <string>` — response body must contain this string
- `--timeout <seconds>` — request timeout (default: 10)

## TCP

Checks that a TCP port is open and accepting connections.

```bash
forge watchforge add \
  --type tcp \
  --name "Postgres" \
  --target localhost:5432 \
  --interval 120
```

## Docker Container

Checks that a named container is running and, if the container has a healthcheck defined, that it is healthy.

```bash
forge watchforge add \
  --type docker \
  --name "hemis app" \
  --target hemis-web \
  --interval 30
```

`--target` is the container name as shown in `docker ps`.

## SSL Certificate Expiry

Checks the TLS certificate for a domain and alerts before it expires.

```bash
forge watchforge add \
  --type ssl \
  --name "hemis SSL" \
  --target hemis.example.com \
  --interval 3600
```

Alert thresholds (not configurable):
- `WARNING` at 14 days remaining
- `CRITICAL` at 3 days remaining

## Heartbeat (Dead Man's Snitch)

WatchForge generates a unique ping URL. The monitored script or cron job must call this URL as its last step. If no ping is received within `interval + grace`, an alert fires.

```bash
forge watchforge add \
  --type heartbeat \
  --name "nightly backup" \
  --interval 86400 \
  --grace 3600
```

Output: a unique URL like `https://status.example.com/_ping/abc123`

Add to your cron job or script:
```bash
# At end of script, after all commands succeed:
curl -s https://status.example.com/_ping/abc123
```
