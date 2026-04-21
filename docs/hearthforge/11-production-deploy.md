# Production Deployment

This document describes the recommended production deployment approach for apps hosted alongside HearthForge.

> For full deployment management, use **SmeltForge**. This document covers the minimal manual approach for when SmeltForge is not installed.

## Principle

Production deployments should be artifact-based, not repository-based:
- Do not `git pull` directly into production services
- Build immutable images in CI
- Deploy by pulling images and restarting services

## Recommended Flow

```
1. Developer pushes to GitHub
2. CI builds Docker images
3. CI pushes images to a registry (e.g. GHCR)
4. VPS pulls updated images
5. VPS restarts services (docker compose)
6. Proxy continues routing to updated containers
```

## Benefits

- Reproducible deploys
- Easier rollback
- No mutable production code directories
- Consistent environments

## VPS Responsibilities

- Store minimal deploy credentials (registry read token) in `forge secrets`
- Run a deploy script that:
  - Pulls images
  - Restarts compose stacks
  - Validates health checks
- Keep production services attached to `web` only

## Secrets Management

Store secrets in `forge secrets`, not in `.env` files:

```bash
forge secrets set myapp.DATABASE_URL "postgres://..."
forge secrets set myapp.REGISTRY_TOKEN "ghp_..."
```

Restrict permissions to admin only.

## Using SmeltForge Instead

If SmeltForge is installed, all of the above is handled automatically:

```bash
forge smeltforge add --project myapp
forge smeltforge deploy --project myapp
```

SmeltForge handles image pulls, container restarts, Caddy proxy routing, and deploy audit logging. See [SmeltForge docs](../smeltforge/README.md) for details.
