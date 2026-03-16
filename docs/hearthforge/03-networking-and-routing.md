# Networking and Routing

This document explains Docker networks, how routing works for production and dev environments, and where vhosts live.

## Networks

Two external Docker networks are used:

- `web` — production services network
- `dev-web` — developer environments network

Rules:
- Production services attach only to `web`
- Developer containers attach only to `dev-web`
- The global proxy (Nginx or Caddy) attaches to both networks

This prevents dev containers from discovering or reaching production services by Docker DNS name.

## Proxy Routing Model

Routing is based on:
- Host header (`server_name`) selects the site (prod or dev)
- Path (`location`) selects the service inside that site

Examples:
- `/` routes to frontend
- `/api` routes to backend
- `/socket.io` routes to backend with upgrade
- `/health` routes to backend health endpoint

## Where Vhosts Live

```
conf.d/examples/    # committed templates and examples
conf.d/active/      # live vhosts on the server (not committed)
```

The proxy compose mounts: `./conf.d/active:/etc/nginx/conf.d`

If SmeltForge is installed, HearthForge registers dev vhosts in SmeltForge's Caddy instance instead of managing its own Nginx.

## Developer Routing (Pattern A)

One dev hostname per developer per project:

```
<dev>-<project>.dev.domain.com
```

Each hostname maps to a single dev container:
- Container name: `dev-<project>-<dev>`
- Preview port(s): defined by project registry

Note: SSH to dev containers is not routed by the proxy. SSH is handled by the gateway on port 2224.

## Avoiding Port Sprawl

- Only proxy exposes 80/443
- Only gateway exposes the SSH port (2224)
- Dev containers use `expose` only (internal to `dev-web`)
- No per-dev host ports

## Recommended Checks

```bash
# Confirm proxy joins both networks
docker network inspect web
docker network inspect dev-web

# Confirm dev containers only on dev-web
docker inspect <container> | grep -A3 Networks

# Validate Nginx config inside proxy container
docker exec -it nginx-proxy nginx -t
```
