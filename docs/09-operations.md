# Operations

This document covers day-2 operations: maintaining proxy, gateway, registries, workspaces, and images.

## Routine operations

### Proxy
- Validate config:

  - `docker exec -it nginx-proxy nginx -t`
- Reload config:
  - `docker exec -it nginx-proxy nginx -s reload`
- Review active vhosts:
  - `/opt/infra/proxy/conf.d/active/`

### Forge
- Update registries (`projects.json`) via `devctl add-project`
- Provision dev envs via `devctl add-dev`
- Review audit logs:
  - `gateway/logs/audit.log`

## Updating dev images

- Rebuild dev base images when:
  - toolchain versions change (node/python)
  - security patches required
- Consider versioned images and pinning to prevent surprise breakage.

## Backups

Decide policy:
- What to back up:
  - `/opt/data/dev_workspaces/` (optional, depending on whether everything is in Git)
  - gateway audit logs
  - proxy configuration (active vhosts)
- How often:
  - daily/weekly for workspaces if needed
- Where:
  - `/opt/data/backups/` and optionally offsite storage

## TLS operations

- Production: certbot renew should run automatically
- Dev wildcard: renew via DNS-01 automation
- Verify renewal:
  - check certificate expiry periodically

## Monitoring

At minimum:
- proxy container health
- gateway process uptime
- disk usage in `/opt/data/dev_workspaces`
- failed SSH auth attempts (gateway logs)

## Common maintenance tasks

- Rotate developer keys
- Disable or delete dormant dev environments
- Prune unused dev images and stopped containers (admin only)
- Keep dev-web and web networks clean and separated

