# SmeltForge

SmeltForge is the deployment platform. It deploys Docker Compose stacks and single containers from Git repos, Docker registries, or local directories. It manages a built-in Caddy reverse proxy for automatic TLS and domain routing.

## What It Does

- Deploys Docker Compose stacks and single containers
- Manages automatic TLS via Caddy + Let's Encrypt
- Routes domains to containers automatically
- Triggers deploys from git push, webhooks, polling, or CI
- Injects secrets from `forge secrets` as environment variables
- Maintains a full deploy audit trail
- Integrates with WatchForge to pause monitors during deploys
- Integrates with PenForge for post-deploy security scans

## Installation

```bash
forge install smeltforge
```

## Quickstart

```bash
# Register a project
forge smeltforge add --project myapp

# Deploy it
forge smeltforge deploy --project myapp

# Check status
forge smeltforge status
```

## Deploy Sources

| Source | Description |
|---|---|
| Git repo | Clones and builds from a GitHub/GitLab/Gitea repo |
| Docker image | Pulls from Docker Hub, GHCR, or a self-hosted registry |
| Local directory | Deploys a Compose stack already on the VPS |

## Deploy Strategies

| Strategy | Downtime | When to use |
|---|---|---|
| Stop → Start (default) | ~5-10 seconds | Personal projects, low traffic |
| Blue-Green | Zero | Production, customer-facing apps |

Set per project. Default is Stop → Start.

## Deploy Triggers

```bash
# Manual
forge smeltforge deploy --project myapp

# Webhook (GitHub/GitLab sends POST on push)
# URL: https://yourdomain.com/_smeltforge/webhook/<project>/<secret>

# CI trigger (from GitHub Actions or any CI)
curl -X POST https://yourdomain.com/_smeltforge/deploy \
  -H "Authorization: Bearer $SMELTFORGE_TOKEN" \
  -d '{"project": "myapp"}'

# Polling (SmeltForge checks repo every N minutes)
# Configured per project in registry/projects.json
```

## Environment Variables

```bash
forge smeltforge env set myapp DATABASE_URL "postgres://..."
forge smeltforge env list myapp       # values redacted
forge smeltforge env unset myapp DATABASE_URL
```

Secrets are stored in `forge secrets` (namespaced as `smeltforge.myapp.<KEY>`) and injected at deploy time. No plaintext `.env` files on disk.

## CLI Reference

```bash
forge smeltforge init
forge smeltforge add --project <id>
forge smeltforge deploy --project <id>
forge smeltforge deploy --project <id> --node <node>   # with FluxForge
forge smeltforge rollback --project <id>
forge smeltforge status
forge smeltforge logs --project <id>
forge smeltforge env set <project> <key> <value>
forge smeltforge env list <project>
forge smeltforge env unset <project> <key>
forge smeltforge webhook regenerate <project>
```

## With FluxForge

When FluxForge is installed, SmeltForge gains:
- `--node <node>` flag on deploy — deploy to any node in the mesh
- Shared project registry synced across nodes
- Env vars synced across nodes via `forge secrets --sync`

## Deep Documentation

See [`smeltforge/docs/`](../../smeltforge/docs/) for:
- Project registry schema reference
- Caddy proxy configuration
- Deployment strategy internals
- Webhook security
- Audit trail format
