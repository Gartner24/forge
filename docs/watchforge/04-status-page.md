# Status Page

WatchForge generates a static HTML public status page served at `https://status.<forge-domain>`.

## Configuration

Monitors appear on the status page only when `--public` is set:

```bash
forge watchforge add --type http --name "API" --target https://api.example.com --public
```

Existing monitor:
```bash
forge watchforge update --monitor <id> --public true
```

## What the Page Shows

- Overall system status banner (all healthy / degraded / major outage)
- Per-monitor status: HEALTHY / DOWN / PAUSED
- 30-day uptime percentage per monitor
- Active incident details
- Incident history (last 90 days by default)

## Static File Serving

The status page is a single `index.html` file written to `data/status/index.html`. Caddy serves it directly. Because it is a static file, it remains accessible even if the WatchForge daemon is restarted or crashes — Caddy serves the last generated version.

The file is regenerated atomically after every check cycle (write to temp → rename into place), so a partial write never corrupts the live page.

## Custom Domain

By default the page is served at `https://status.<forge-domain>`. To use a different domain, configure it in `config.toml`:

```toml
[modules.watchforge]
status_page_url = "https://status.mycustomdomain.com"
```

Ensure the domain's DNS points to your server and Caddy has a matching route configured.
