# Troubleshooting

This document provides a checklist for common issues.

## VS Code / Cursor cannot connect

Checklist:
- Does the target hostname resolve to the VPS IP?
- Is the gateway reachable on the SSH port?
- Is the dev container running?
- Is `sshd` running inside the dev container?
- Is the developer key present and mapped correctly?
- Is the container reachable from the gateway (network `dev-web`)?

Useful checks:
- `docker ps | grep dev-`
- `docker exec -it <dev-container> ps aux | grep sshd`
- `docker network inspect dev-web`

## SSH works but no shell / permission issues

- Confirm the container user exists and has a home directory.
- Confirm `sshd_config` allows the intended user.
- Confirm workspace permissions under `/workspace/<project>`.

## Proxy routes to wrong target

- Verify vhost file exists in `proxy/conf.d/active/`.
- Validate proxy config:
  - `docker exec -it nginx-proxy nginx -t`
- Check for duplicate `server_name` collisions.

## ACME / cert renewal issues

- Confirm port 80 is open on the VPS.
- Confirm `/.well-known/acme-challenge/` is routed to the webroot directory.
- Confirm certbot container is running and has correct volumes.

## Container DNS name not resolving

- Confirm proxy and target container share the same network.
- For dev routing, confirm both are on `dev-web`.
- For prod routing, confirm both are on `web`.

## Dev container cannot access internet / git

- Confirm container has outbound connectivity.
- Confirm DNS resolution works inside container.
- If corporate/proxy restrictions apply, configure accordingly.

## Disk usage issues

- Check `/opt/data/dev_workspaces` size.
- Prune unused containers/images (admin only).
- Implement workspace retention policy.

## Quick verification commands

- Proxy config test:
  - `docker exec -it nginx-proxy nginx -t`
- Proxy networks:
  - `docker inspect nginx-proxy | grep -A6 Networks`
- Dev container networks:
  - `docker inspect dev-<project>-<dev> | grep -A6 Networks`
- Dev container sshd:
  - `docker exec -it dev-<project>-<dev> ss -lntp | grep :22`

