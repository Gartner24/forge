# Production deployment

This document describes the recommended production deployment approach for apps hosted on the VPS.

## Principle

Production deployments should be artifact-based, not repository-based:
- do not `git pull` directly into production services
- build immutable images in CI
- deploy by pulling images and restarting services

## Recommended flow


1. Developer pushes to GitHub
2. CI builds Docker images
3. CI pushes images to a registry (e.g., GHCR)
4. VPS pulls updated images
5. VPS restarts services (docker compose)
6. Proxy continues routing to the updated containers

## Benefits

- reproducible deploys
- easier rollback
- no mutable production code directories
- consistent environments

## VPS responsibilities

- store minimal deploy credentials (registry read token)
- run a deploy script that:
  - pulls images
  - restarts compose stacks
  - validates health checks
- keep production services attached to `web` only

## Secrets management

- store secrets in `.env` files on the VPS (not committed)
- keep `.env.example` in repos
- restrict permissions to admin only
- consider Docker secrets later if needed

