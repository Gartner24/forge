# Operations

This document covers day-2 operations for HearthForge: maintaining proxy, gateway, registries, workspaces, and images.

## Routine Operations

### Proxy

When SmeltForge is installed (standard setup), HearthForge uses Caddy. Vhosts are managed via the Caddy Admin API — never edit Caddy config files directly.

```bash
# Check Caddy is running
docker ps | grep caddy

# View Caddy logs
forge logs smeltforge

# Force-reload Caddy config (should not normally be needed)
curl -s http://localhost:2019/load
```

Without SmeltForge (standalone Nginx):
```bash
# Validate Nginx config
docker exec -it nginx-proxy nginx -t

# Reload Nginx config
docker exec -it nginx-proxy nginx -s reload
```

### Gateway

```bash
# Review audit logs
sudo tail -n 100 /opt/infra/forge/hearthforge/gateway/logs/audit.log

# Filter by result
sudo rg "result=accepted" /opt/infra/forge/hearthforge/gateway/logs/audit.log
sudo rg "result=rejected" /opt/infra/forge/hearthforge/gateway/logs/audit.log

# Runtime gateway events
sudo docker logs proxy-gateway-1 --tail=80
```

### HearthForge Registry

```bash
# Add or update a project
forge hearthforge add-project

# Provision a dev environment
forge hearthforge add-dev

# List all developers
forge hearthforge list-devs
```

## Updating Dev Images

Rebuild dev base images when:
- Toolchain versions change (Node/Python)
- Security patches are required
- New IDE support is added

Consider versioned images and pinning to prevent surprise breakage.

## Backups

Decide on a backup policy covering:

| What | Suggested frequency | Where |
|---|---|---|
| `/opt/data/dev_workspaces/` | Daily/weekly (if not all in Git) | `/opt/data/backups/` + offsite |
| Gateway audit logs | Keep indefinitely | `/opt/data/logs/gateway/` |
| Proxy active vhosts | Weekly | `/opt/data/backups/` |

## Long-Term Log Retention

Recommended structure:
```
/opt/data/logs/
  gateway/
    audit.log
  proxy/
    access.log
    error.log
```

Point gateway audit logs to `/opt/data/logs/gateway/` in `gateway.toml`:
```toml
[paths]
audit_log_dir = "/opt/data/logs/gateway"
```

## TLS Operations

- Production: certbot renews automatically
- Dev wildcard: renew via DNS-01 automation (Cloudflare API)
- Verify renewal: check certificate expiry periodically

If SmeltForge/Caddy is used, TLS renewal is fully automatic — no manual action needed.

## Monitoring

Install WatchForge for automated monitoring. Without WatchForge, check at minimum:
- Proxy container health
- Gateway process uptime
- Disk usage in `/opt/data/dev_workspaces/`
- Failed SSH auth attempts (gateway logs)

## Common Maintenance Tasks

- Rotate developer keys: `forge hearthforge gateway-add-key --dev <dev>`
- Disable dormant environments: `forge hearthforge delete-dev --dev <dev> --all-projects`
- Prune unused dev images and stopped containers (admin only): `docker image prune`, `docker container prune`
- Keep `dev-web` and `web` networks clean and separated
